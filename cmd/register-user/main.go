package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"

	"hanashite/cmd/server/channel/users"
)

func main() {
	var (
		username = flag.String("user", "", "Username to register")
		dataDir  = flag.String("data", "./data", "Data directory for user storage")
	)
	flag.Parse()

	if *username == "" {
		fmt.Println("Usage: register-user -user <username> [-data <data-dir>]")
		flag.Usage()
		os.Exit(1)
	}

	userService, err := users.NewUserService(*dataDir)
	if err != nil {
		log.Fatalf("Failed to create user service: %v", err)
	}

	x25519PrivateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	publicKey := x25519PrivateKey.PublicKey().Bytes()
	privateKey := x25519PrivateKey.Bytes()

	err = userService.RegisterUser(*username, publicKey)
	if err != nil {
		log.Fatalf("Failed to register user: %v", err)
	}

	fmt.Printf("User '%s' registered successfully!\n", *username)
	fmt.Printf("Public Key: %x\n", publicKey)
	fmt.Printf("Private Key: %x\n", privateKey)
	fmt.Printf("\nSave the private key securely for the client to use.\n")
}
