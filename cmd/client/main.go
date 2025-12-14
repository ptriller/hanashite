package main

import (
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	v1 "hanashite/api/v1"
	"hanashite/internal/common"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	clientInit = iota
	clientWaitChallenge
	clientWaitResponse
	clientAuthenticated
)

type Client struct {
	state     int
	conn      net.Conn
	timeout   time.Duration
	msgBuf    []byte
	aead      cipher.AEAD
	challenge []byte
	clientKey *ecdh.PrivateKey
	serverKey *ecdh.PublicKey
	authKey   ed25519.PrivateKey
}

type ClientConfig struct {
	Users []ClientUser `json:"users"`
}

type ClientUser struct {
	Name       string `json:"name"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func LoadClientConfig(configPath string) (*ClientConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read client config: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse client config: %w", err)
	}

	return &config, nil
}

func (cc *ClientConfig) GetUserPrivateKey(name string) (string, error) {
	for _, user := range cc.Users {
		if user.Name == name {
			return user.PrivateKey, nil
		}
	}
	return "", fmt.Errorf("user %s not found in client config", name)
}

func NewClientWithKeys(privateKeyHex string) (*Client, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(privateKeyBytes) != 32 {
		return nil, fmt.Errorf("invalid private key length, expected 32 bytes, got %d", len(privateKeyBytes))
	}

	x25519PrivateKey, err := ecdh.X25519().NewPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create X25519 private key: %w", err)
	}

	authKey := ed25519.NewKeyFromSeed(privateKeyBytes)

	return &Client{
		state:     clientInit,
		timeout:   time.Second * 5,
		clientKey: x25519PrivateKey,
		authKey:   authKey,
	}, nil
}

func (c *Client) Connect(address string) error {
	zap.L().Info("Attempting to connect to server", zap.String("address", address))

	conn, err := net.DialTimeout("tcp", address, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	c.conn = conn
	zap.L().Info("TCP connection established")

	go c.handleConnection()
	return nil
}

func (c *Client) handleConnection() {
	for {
		msg, encrypted, err := c.readMessage()
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			zap.L().Info("Message timeout received")
			continue
		} else if err != nil {
			zap.S().Warn("Error on network connection", zap.Error(err))
			_ = c.conn.Close()
			return
		}

		if c.state != clientInit && !encrypted {
			zap.L().Warn("Unencrypted message after init")
		}

		switch c.state {
		case clientInit:
			c.handleInitResponse(msg)
		case clientWaitChallenge:
			c.handleChallenge(msg)
		case clientWaitResponse:
			c.handleAuthResponse(msg)
		case clientAuthenticated:
			c.handleAuthenticatedMessage(msg)
		default:
			zap.L().Warn("Unknown state", zap.Int("state", c.state))
		}
	}
}

func (c *Client) handleInitResponse(msg *v1.Envelope) {
	if msg.Type != v1.PayloadType_TConnectResponse {
		zap.L().Error("Unexpected message type in init state", zap.Int("type", int(msg.Type)))
		return
	}

	var response v1.ConnectResponse
	err := c.decodeMessage(msg.Msg, &response)
	if err != nil {
		return
	}

	serverKey, err := ecdh.X25519().NewPublicKey(response.ServerKey)
	if err != nil {
		zap.L().Error("Failed to parse server public key", zap.Error(err))
		return
	}
	c.serverKey = serverKey

	sharedSecret, err := c.clientKey.ECDH(c.serverKey)
	if err != nil {
		zap.L().Error("Failed to compute shared secret", zap.Error(err))
		return
	}

	aead, err := chacha20poly1305.New(sharedSecret)
	if err != nil {
		zap.L().Error("Failed to create AEAD", zap.Error(err))
		return
	}
	c.aead = aead

	c.state = clientWaitChallenge
	zap.L().Info("Key exchange completed, waiting for challenge")
}

func (c *Client) handleChallenge(msg *v1.Envelope) {
	if msg.Type != v1.PayloadType_TLoginChallenge {
		zap.L().Error("Unexpected message type in wait challenge state", zap.Int("type", int(msg.Type)))
		return
	}

	var challenge v1.LoginChallenge
	err := c.decodeMessage(msg.Msg, &challenge)
	if err != nil {
		return
	}

	c.challenge = challenge.Challenge

	response := c.calculateChallengeResponse()
	err = c.sendMessage(v1.PayloadType_TLoginRequest, &v1.LoginRequest{
		ChallengeResponse: response,
	})
	if err != nil {
		zap.L().Error("Failed to send login request", zap.Error(err))
		return
	}

	c.state = clientWaitResponse
	zap.L().Info("Challenge response sent, waiting for authentication result")
}

func (c *Client) calculateChallengeResponse() []byte {
	signature := ed25519.Sign(c.authKey, c.challenge)
	return signature
}

func (c *Client) handleAuthResponse(msg *v1.Envelope) {
	if msg.Type == v1.PayloadType_TError {
		var errorResp v1.ErrorResponse
		err := c.decodeMessage(msg.Msg, &errorResp)
		if err != nil {
			zap.L().Error("Failed to decode error response", zap.Error(err))
			return
		}
		// Check if this is actually a success message
		if strings.Contains(errorResp.Message, "Authentication successful") || strings.Contains(errorResp.Message, "Welcome") {
			c.state = clientAuthenticated
			zap.L().Info("Authentication successful!", zap.String("message", errorResp.Message))
			return
		}
		zap.L().Error("Authentication failed", zap.String("error", errorResp.Message))
		return
	}

	c.state = clientAuthenticated
	zap.L().Info("Authentication successful!")
}

func (c *Client) handleAuthenticatedMessage(msg *v1.Envelope) {
	zap.L().Info("Received authenticated message", zap.Int("type", int(msg.Type)))
}

func (c *Client) StartConnection() error {
	zap.L().Info("Starting connection process")

	err := c.sendMessage(v1.PayloadType_TConnectRequest, &v1.ConnectRequest{
		ClientKey: c.clientKey.PublicKey().Bytes(),
	})
	if err != nil {
		return fmt.Errorf("failed to send connect request: %w", err)
	}

	c.state = clientInit
	zap.L().Info("Connection request sent", zap.Int("state", c.state))
	return nil
}

func (c *Client) sendMessage(payloadType v1.PayloadType, message proto.Message) error {
	msg, err := c.createMessage(payloadType, message)
	if err != nil {
		return fmt.Errorf("error creating message: %w", err)
	}

	if c.aead != nil {
		nonce := make([]byte, c.aead.NonceSize())
		_, err = rand.Read(nonce)
		if err != nil {
			return fmt.Errorf("error creating nonce: %w", err)
		}
		data := c.aead.Seal(nil, nonce, msg, nil)
		msg, err = proto.Marshal(&v1.Envelope{
			Nonce: nonce,
			Msg:   data,
		})
		if err != nil {
			return fmt.Errorf("error marshalling envelope: %w", err)
		}
	}

	// Write length prefix (4 bytes big endian)
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(msg)))

	_, err = c.conn.Write(lenBuf)
	if err != nil {
		return fmt.Errorf("error writing length: %w", err)
	}

	length, err := c.conn.Write(msg)
	if err != nil {
		return fmt.Errorf("error sending message: %w", err)
	}
	if length != len(msg) {
		return fmt.Errorf("error sending message: expected %d bytes, got %d", len(msg), length)
	}
	return nil
}

func (c *Client) createMessage(payloadType v1.PayloadType, message proto.Message) ([]byte, error) {
	data, err := proto.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("error marshaling message: %w", err)
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

func (c *Client) decodeMessage(buffer []byte, message proto.Message) error {
	err := proto.Unmarshal(buffer, message)
	if err != nil {
		return fmt.Errorf("error decoding message: %w", err)
	}
	return nil
}

func (c *Client) readMessage() (*v1.Envelope, bool, error) {
	var lenBuf [4]byte
	err := c.readBuf(lenBuf[:])
	if err != nil {
		return nil, false, err
	}
	msgLen := binary.BigEndian.Uint32(lenBuf[:])
	if c.msgBuf == nil || uint32(len(c.msgBuf)) < msgLen {
		c.msgBuf = make([]byte, msgLen)
	}
	msg := c.msgBuf[:msgLen]
	err = c.readBuf(msg)
	if err != nil {
		_ = c.conn.Close()
		return nil, false, fmt.Errorf("error reading message after msgsize: %w", err)
	}
	return c.unfoldMessage(msg)
}

func (c *Client) unfoldMessage(msg []byte) (*v1.Envelope, bool, error) {
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
		env, _, err := c.unfoldMessage(decrypt)
		return env, true, err
	}
	return &protoMessage, false, nil
}

func (c *Client) readBuf(buf []byte) error {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout)); err != nil {
		return err
	}
	num, err := io.ReadFull(c.conn, buf)
	if err != nil {
		return err
	}
	if num != len(buf) {
		_ = c.conn.Close()
		return errors.New("impossible ReadFull call")
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: client <server-address> <username>")
		fmt.Println("Example: client localhost:6543 testuser")
		os.Exit(1)
	}

	// Initialize logger
	common.PreLogger()

	serverAddr := os.Args[1]
	username := os.Args[2]

	// Load client configuration
	config, err := LoadClientConfig("data/clients.json")
	if err != nil {
		zap.S().Fatalf("Failed to load client config: %v", err)
	}

	privateKeyHex, err := config.GetUserPrivateKey(username)
	if err != nil {
		zap.S().Fatalf("Failed to get private key for user %s: %v", username, err)
	}

	client, err := NewClientWithKeys(privateKeyHex)
	if err != nil {
		zap.S().Fatalf("Failed to create client: %v", err)
	}

	fmt.Printf("Client X25519 public:  %x\n", client.clientKey.PublicKey().Bytes())

	err = client.Connect(serverAddr)
	if err != nil {
		zap.S().Fatalf("Failed to connect to server: %v", err)
	}

	err = client.StartConnection()
	if err != nil {
		zap.S().Fatalf("Failed to start connection: %v", err)
	}

	// Wait for authentication or timeout
	time.Sleep(10 * time.Second)
	zap.L().Info("Client exiting after timeout")
}
