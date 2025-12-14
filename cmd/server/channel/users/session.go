package users

import (
	"sync"
	"time"
)

type Session struct {
	ID           string    `json:"id"`
	UserName     string    `json:"user_name"`
	PublicKey    string    `json:"public_key"`
	ConnectedAt  time.Time `json:"connected_at"`
	LastActivity time.Time `json:"last_activity"`
}

type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

func (sm *SessionManager) CreateSession(sessionID, userName, publicKey string) *Session {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	session := &Session{
		ID:           sessionID,
		UserName:     userName,
		PublicKey:    publicKey,
		ConnectedAt:  now,
		LastActivity: now,
	}

	sm.sessions[sessionID] = session
	return session
}

func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

func (sm *SessionManager) UpdateActivity(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.LastActivity = time.Now()
	}
}

func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	delete(sm.sessions, sessionID)
}

func (sm *SessionManager) GetSessionsByUser(userName string) []*Session {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var userSessions []*Session
	for _, session := range sm.sessions {
		if session.UserName == userName {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions
}

func (sm *SessionManager) CleanupInactiveSessions(timeout time.Duration) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastActivity) > timeout {
			delete(sm.sessions, sessionID)
		}
	}
}

func (sm *SessionManager) GetAllSessions() []*Session {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}
