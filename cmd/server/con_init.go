package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"fmt"
	v1 "hanashite/api/v1"

	"golang.org/x/crypto/chacha20poly1305"
)

func (c *Connection) HandleInitConnection(msg *v1.Envelope) {
	if msg.Type != v1.PayloadType_TConnectRequest {
		c.HandleErrorAndClose(fmt.Sprintf("Unexpected Message in Init State: %d", msg.Type))
		return
	}
	var request v1.ConnectRequest
	err := DecodeMessage(c, msg.Msg, &request)
	if err != nil {
		return
	}
	clientKey, err := ecdh.X25519().NewPublicKey(request.ClientKey)
	if err != nil {
		c.HandleErrorAndClose(fmt.Sprintf("Error Public key from Client: %s", err))
		return
	}
	c.clientKey = clientKey
	cryptKey, err := c.serverKey.ECDH(c.clientKey)
	if err != nil {
		c.HandleErrorAndClose(fmt.Sprintf("Error Generating Shared Crypt Key: %s", err))
		return
	}
	aead, err := chacha20poly1305.New(cryptKey)
	if err != nil {
		c.HandleErrorAndClose(fmt.Sprintf("Error Generating Chacha Key: %s", err))
		return
	}
	err = SendMessage(c, v1.PayloadType_TConnectResponse, &v1.ConnectResponse{
		ServerKey: c.serverKey.PublicKey().Bytes(),
	})
	if err != nil {
		return
	}
	c.aead = aead
	c.challenge = make([]byte, 32)

	_, err = rand.Read(c.challenge)
	if err != nil {
		c.HandleErrorAndClose(fmt.Sprintf("Error Generating Challenge: %s", err))
		return
	}
	err = SendMessage(c, v1.PayloadType_TLoginChallenge, &v1.LoginChallenge{
		Challenge: c.challenge,
	})
	if err != nil {
		return
	}
}
