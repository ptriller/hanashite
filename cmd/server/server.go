package main

import (
	"net"

	"go.uber.org/zap"
)

type ServerConfig struct {
	BindAddress string `yaml:"bind-address"`
}

type SocketServer struct {
	address string
}

func NewSocketServer(cfg *ServerConfig) *SocketServer {
	return &SocketServer{
		address: cfg.BindAddress,
	}
}

func (s *SocketServer) Start() {
	// 1. Start listening on the specified network address (TCP)
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		zap.S().Fatalf("❌ Error starting listener: %v", err)
	}
	// Ensure the listener is closed when the main function exits
	defer listener.Close()

	zap.S().Infof("✅ TCP Server listening on %s\n", s.address)

	// 2. Loop forever, accepting incoming connections
	for {
		// listener.Accept() blocks until a new connection is established
		conn, err := listener.Accept()
		if err != nil {
			zap.S().Infof("⚠️ Error accepting connection: %v", err)
			continue
		}

		// 3. Crucial Step: Start a new goroutine to handle the connection.
		// This makes the Accept loop immediately ready for the next client.
		go s.handleConnection(conn)
	}
}

func (s *SocketServer) handleConnection(conn net.Conn) {

}
