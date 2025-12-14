package main

import (
	"net"

	"hanashite/cmd/server/channel/users"

	"go.uber.org/zap"
)

type ServerConfig struct {
	BindAddress string `yaml:"bind-address"`
}

type SocketServer struct {
	address        string
	userService    *users.UserService
	sessionManager *users.SessionManager
}

func NewSocketServer(cfg *ServerConfig) *SocketServer {
	userService, err := users.NewUserService("./data")
	if err != nil {
		zap.S().Fatalf("Failed to create user service: %v", err)
	}

	sessionManager := users.NewSessionManager()

	return &SocketServer{
		address:        cfg.BindAddress,
		userService:    userService,
		sessionManager: sessionManager,
	}
}

func (s *SocketServer) Start() {
	// 1. Start listening on the specified network address (TCP)
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		zap.S().Fatalf("❌ Error starting listener: %v", err)
	}
	// Ensure the listener is closed when the main function exits
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			zap.S().Fatalf("Error closing listener: %v", err)
		}
	}(listener)

	zap.S().Infof("✅ TCP Server listening on %s\n", s.address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			zap.S().Infof("⚠️ Error accepting connection: %v", err)
			continue
		}
		s.handleConnection(conn)
	}
}

func (s *SocketServer) handleConnection(conn net.Conn) {
	connection, err := NewConnection(conn, s.userService, s.sessionManager)
	if err != nil {
		zap.S().Warnf("Unable to establish connection: %v", err)
		return
	}
	go connection.HandleConnection()
}
