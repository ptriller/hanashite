package users

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
)

type UserService struct {
	usersFile string
	users     map[string]*User
	mutex     sync.RWMutex
}

func NewUserService(dataDir string) (*UserService, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	usersFile := filepath.Join(dataDir, "users.json")
	service := &UserService{
		usersFile: usersFile,
		users:     make(map[string]*User),
	}

	if err := service.loadUsers(); err != nil {
		zap.L().Warn("Failed to load users, starting with empty user database", zap.Error(err))
	}

	return service, nil
}

func (us *UserService) loadUsers() error {
	us.mutex.Lock()
	defer us.mutex.Unlock()

	data, err := os.ReadFile(us.usersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var users []*User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	for _, user := range users {
		us.users[user.Name] = user
	}

	zap.L().Info("Loaded users", zap.Int("count", len(us.users)))
	return nil
}

func (us *UserService) saveUsers() error {
	us.mutex.RLock()
	users := make([]*User, 0, len(us.users))
	for _, user := range us.users {
		users = append(users, user)
	}
	us.mutex.RUnlock()

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	if err := os.WriteFile(us.usersFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}

func (us *UserService) RegisterUser(name string, x25519PublicKey []byte) error {
	us.mutex.Lock()

	if _, exists := us.users[name]; exists {
		us.mutex.Unlock()
		return fmt.Errorf("user %s already exists", name)
	}

	user := &User{
		Name:      name,
		PublicKey: hex.EncodeToString(x25519PublicKey),
	}

	us.users[name] = user
	us.mutex.Unlock()

	if err := us.saveUsers(); err != nil {
		us.mutex.Lock()
		delete(us.users, name)
		us.mutex.Unlock()
		return fmt.Errorf("failed to save user: %w", err)
	}

	zap.L().Info("User registered",
		zap.String("name", name),
		zap.String("x25519_public", hex.EncodeToString(x25519PublicKey)))
	return nil
}

func (us *UserService) GetUser(name string) (*User, error) {
	us.mutex.RLock()
	defer us.mutex.RUnlock()

	user, exists := us.users[name]
	if !exists {
		return nil, fmt.Errorf("user %s not found", name)
	}

	return user, nil
}

func (us *UserService) GetUserByPublicKey(publicKey []byte) (*User, error) {
	us.mutex.RLock()
	defer us.mutex.RUnlock()

	pubKeyHex := hex.EncodeToString(publicKey)
	for _, user := range us.users {
		if user.PublicKey == pubKeyHex {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user with public key %s not found", pubKeyHex)
}

func (us *UserService) VerifyChallengeResponse(user *User, challenge, response []byte) error {
	// Validate inputs
	if user == nil {
		return fmt.Errorf("user cannot be nil")
	}
	if len(challenge) == 0 {
		return fmt.Errorf("challenge cannot be empty")
	}
	if len(response) == 0 {
		return fmt.Errorf("signature response cannot be empty")
	}
	if len(response) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: expected %d bytes, got %d", ed25519.SignatureSize, len(response))
	}

	// Get user's Ed25519 public key for signature verification
	publicKeyBytes, err := hex.DecodeString(user.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length: expected %d bytes, got %d", ed25519.PublicKeySize, len(publicKeyBytes))
	}

	// Verify challenge was signed with user's Ed25519 private key
	if !ed25519.Verify(publicKeyBytes, challenge, response) {
		return fmt.Errorf("invalid signature: challenge was not signed with correct private key")
	}

	zap.L().Info("Signature verification successful",
		zap.String("user", user.Name))
	return nil
}

func (us *UserService) ListUsers() ([]*User, error) {
	us.mutex.RLock()
	defer us.mutex.RUnlock()

	users := make([]*User, 0, len(us.users))
	for _, user := range us.users {
		users = append(users, user)
	}
	return users, nil
}
