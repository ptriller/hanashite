package main

import (
	"crypto/rand"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

func main() {
	// Private keys are random 32-byte numbers
	var priv1, priv2 [32]byte
	rand.Read(priv1[:])
	rand.Read(priv2[:])
	// Public keys = X25519 basepoint * private key
	pub1, _ := curve25519.X25519(priv1[:], curve25519.Basepoint)
	pub2, _ := curve25519.X25519(priv2[:], curve25519.Basepoint)

	fmt.Printf("Client private: %x\n", priv1)
	fmt.Printf("Client public:  %x\n", pub1)

	fmt.Printf("Server private: %x\n", priv2)
	fmt.Printf("Server public:  %x\n", pub2)
	Hello
}
