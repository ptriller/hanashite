package main

import (
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	v1 "hanashite/api/v1"
	"io"
	"net"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	conInit = iota
	conWaitCryptAcc
	conIdle
)

type Connection struct {
	state     int
	conn      net.Conn
	timeout   time.Duration
	lastSend  time.Time
	lastRecv  time.Time
	msgBuf    []byte
	aead      cipher.AEAD
	challenge []byte
	clientKey *ecdh.PublicKey
	serverKey *ecdh.PrivateKey
}

func NewConnection(conn net.Conn) (*Connection, error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return &Connection{
		state:     conInit,
		timeout:   time.Second * 5,
		conn:      conn,
		serverKey: key,
	}, nil

}

func (c *Connection) HandleConnection() {
	for {
		msg, encrypted, err := c.ReadMessage()
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			zap.L().Info("Msg Timeout recieved.")
		} else if err != nil {
			zap.S().Warn("Error on Network Connection", zap.Error(err))
			_ = c.conn.Close()
			return
		}
		if c.state != conInit && !encrypted {
			zap.L().Warn("Unencrypted message after Init.")
		}
		switch c.state {
		case conInit:
			c.HandleInitConnection(msg)
		default:
			zap.L().Warn("Unknown State", zap.Int("State", c.state))

		}
	}
}

func SendMessage[T proto.Message](c *Connection, payloadType v1.PayloadType, message T) error {
	msg, err := createMessage(c, payloadType, message)
	if err != nil {
		c.HandleErrorAndClose(fmt.Sprintf("Error creating message: %v", err))
		return err
	}
	if c.aead != nil {
		nonce := make([]byte, c.aead.NonceSize())
		_, err = rand.Read(nonce)
		if err != nil {
			c.HandleErrorAndClose(fmt.Sprintf("Error creating nonce: %v", err))
			return err
		}
		data := c.aead.Seal(nil, nonce, msg, nil)
		msg, err = proto.Marshal(&v1.Envelope{
			Nonce: nonce,
			Msg:   data,
		})
		if err != nil {
			c.HandleErrorAndClose(fmt.Sprintf("Error marshalling envelope: %v", err))
			return err
		}
	}
	length, err := c.conn.Write(msg)
	if err != nil {
		c.HandleErrorAndClose(fmt.Sprintf("Error sending message: %v", err))
		return err
	}
	if length != len(msg) {
		c.HandleErrorAndClose(fmt.Sprintf("Error sending message: expected %d bytes, got %d", len(msg), length))
		return fmt.Errorf("error sending message: expected %d bytes, got %d", len(msg), length)
	}
	return nil
}

func (c *Connection) HandleErrorAndClose(message string) {
	zap.S().Warn("Closing connection with error: %s", message)
	msg, err := createMessage(c, v1.PayloadType_TError, &v1.ErrorResponse{
		Message: message,
	})
	if err != nil {
		zap.S().Warn("Error on Network Connection", zap.Error(err))
	} else {
		_, _ = c.conn.Write(msg)
	}
	_ = c.conn.Close()
}

func DecodeMessage[T proto.Message](c *Connection, buffer []byte, message T) error {
	err := proto.Unmarshal(buffer, message)
	if err != nil {
		c.HandleErrorAndClose(fmt.Sprintf("Error decoding message: %v", err))
		return err
	}
	return nil
}

func (c *Connection) ReadMessage() (*v1.Envelope, bool, error) {
	var lenBuf [4]byte
	err := c.ReadBuf(lenBuf[:])
	if err != nil {
		return nil, false, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	if c.msgBuf == nil || uint32(len(c.msgBuf)) < msgLen {
		c.msgBuf = make([]byte, msgLen)
	}
	msg := c.msgBuf[:msgLen]
	err = c.ReadBuf(msg)
	if err != nil {
		_ = c.conn.Close()
		zap.S().Warnf("Error reading message after msgsize: %s", err.Error())
		return nil, false, errors.New("error reading message after msgsize")
	}
	return c.UnfoldMessage(msg)
}

func (c *Connection) UnfoldMessage(msg []byte) (*v1.Envelope, bool, error) {
	var protoMessage v1.Envelope
	err := proto.Unmarshal(msg, &protoMessage)
	if err != nil {
		return nil, false, err
	}
	if protoMessage.Nonce != nil {
		decrypt, err := c.aead.Open(nil, protoMessage.Nonce, protoMessage.Msg, nil)
		if err != nil {
			return nil, true, err
		}
		env, _, err := c.UnfoldMessage(decrypt)
		return env, true, err
	}
	return &protoMessage, false, nil
}

func (c *Connection) ReadBuf(buf []byte) error {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	num, err := io.ReadFull(c.conn, buf)
	if err != nil {
		return err
	}
	if num != len(buf) {
		// Impossible error. Close connection
		_ = c.conn.Close()
		return errors.New("impossible ReadFull call")
	}
	return nil
}

func createMessage[T proto.Message](c *Connection, payloadType v1.PayloadType, message T) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		zap.S().Warn("Error marshaling message", zap.Error(err))
		return nil, err
	}
	result, err := proto.Marshal(&v1.Envelope{
		Type: payloadType,
		Msg:  data,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
