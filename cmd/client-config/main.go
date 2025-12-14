package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"go.uber.org/zap"
)

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

func (cc *ClientConfig) GetUserPublicKey(name string) (string, error) {
	for _, user := range cc.Users {
		if user.Name == name {
			return user.PublicKey, nil
		}
	}
	return "", fmt.Errorf("user %s not found in client config", name)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: client-config <username>")
		fmt.Println("Example: client-config testuser")
		os.Exit(1)
	}

	username := os.Args[1]

	config, err := LoadClientConfig("data/clients.json")
	if err != nil {
		zap.S().Fatalf("Failed to load client config: %v", err)
	}

	privateKey, err := config.GetUserPrivateKey(username)
	if err != nil {
		zap.S().Fatalf("Failed to get private key for user %s: %v", username, err)
	}

	publicKey, err := config.GetUserPublicKey(username)
	if err != nil {
		zap.S().Fatalf("Failed to get public key for user %s: %v", username, err)
	}

	fmt.Printf("Private Key: %s\n", privateKey)
	fmt.Printf("Public Key:  %s\n", publicKey)

	// Validate private key format
	privateKeyBytes, err := hex.DecodeString(privateKey)
	if err != nil {
		zap.S().Fatalf("Invalid private key format: %v", err)
	}
	if len(privateKeyBytes) != 32 {
		zap.S().Fatalf("Invalid private key length: expected 32 bytes, got %d", len(privateKeyBytes))
	}

	// Validate public key format
	publicKeyBytes, err := hex.DecodeString(publicKey)
	if err != nil {
		zap.S().Fatalf("Invalid public key format: %v", err)
	}
	if len(publicKeyBytes) != 32 {
		zap.S().Fatalf("Invalid public key length: expected 32 bytes, got %d", len(publicKeyBytes))
	}

	zap.L().Info("Client keys validated successfully")
}
