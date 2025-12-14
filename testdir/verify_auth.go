package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run testdir/verify_auth.go <private-key>")
		fmt.Println("Verifies that only users with correct private key can authenticate")
		os.Exit(1)
	}

	privateKey := os.Args[1]

	fmt.Println("🔐 Testing authentication verification...")

	// Start server
	fmt.Println("🚀 Starting server...")
	serverCmd := exec.Command("../bin/server")
	serverCmd.Env = append(os.Environ(), "CONFIG_FILE=../config.yml")
	if err := serverCmd.Start(); err != nil {
		fmt.Printf("❌ Failed to start server: %v\n", err)
		return
	}

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Test authentication
	fmt.Println("🔑 Testing authentication...")
	clientCmd := exec.Command("../bin/client", "localhost:6543", privateKey)
	clientCmd.Env = os.Environ()

	output, err := clientCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Client failed: %v\n", err)
	} else {
		fmt.Printf("Client output: %s\n", string(output))

		// Check if authentication was successful
		if strings.Contains(string(output), "Authentication successful") || strings.Contains(string(output), "authenticated successfully") {
			fmt.Println("✅ AUTHENTICATION VERIFICATION PASSED!")
			fmt.Println("   User successfully authenticated with correct private key")
		} else if strings.Contains(string(output), "Authentication failed") || strings.Contains(string(output), "user not found") {
			fmt.Println("❌ AUTHENTICATION VERIFICATION FAILED!")
			fmt.Println("   User was rejected - this should not happen with correct key")
		} else {
			fmt.Println("❌ AUTHENTICATION VERIFICATION INCONCLUSIVE!")
			fmt.Printf("   Unexpected result: %s\n", string(output))
		}
	}

	// Stop server
	fmt.Println("🛑 Stopping server...")
	_ = serverCmd.Process.Kill()
	_ = serverCmd.Wait()
}
