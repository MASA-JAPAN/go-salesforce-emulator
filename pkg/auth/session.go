package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session represents an authenticated session
type Session struct {
	AccessToken  string    `json:"access_token"`
	InstanceURL  string    `json:"instance_url"`
	TokenType    string    `json:"token_type"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"-"`
	OrgID        string    `json:"-"`
}

// IsValid checks if the session is still valid
func (s *Session) IsValid() bool {
	return time.Now().Before(s.ExpiresAt)
}

// SessionManager manages active sessions
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	lifetime time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(lifetime time.Duration) *SessionManager {
	if lifetime == 0 {
		lifetime = 2 * time.Hour
	}
	return &SessionManager{
		sessions: make(map[string]*Session),
		lifetime: lifetime,
	}
}

// CreateSession creates a new session
func (m *SessionManager) CreateSession(instanceURL, userID, orgID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	token := generateToken()
	now := time.Now()

	session := &Session{
		AccessToken:  token,
		InstanceURL:  instanceURL,
		TokenType:    "Bearer",
		IssuedAt:     now,
		ExpiresAt:    now.Add(m.lifetime),
		UserID:       userID,
		OrgID:        orgID,
	}

	m.sessions[token] = session
	return session
}

// GetSession retrieves a session by access token
func (m *SessionManager) GetSession(accessToken string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[accessToken]
	if !ok {
		return nil, false
	}

	if !session.IsValid() {
		return nil, false
	}

	return session, true
}

// InvalidateSession removes a session
func (m *SessionManager) InvalidateSession(accessToken string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, accessToken)
}

// CleanExpired removes all expired sessions
func (m *SessionManager) CleanExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for token, session := range m.sessions {
		if !session.IsValid() {
			delete(m.sessions, token)
		}
	}
}

// generateToken generates a random access token
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
