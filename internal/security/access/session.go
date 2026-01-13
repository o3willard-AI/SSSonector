package access

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SessionState represents the state of a session
type SessionState int

const (
	SessionStateActive SessionState = iota
	SessionStateExpired
	SessionStateRevoked
	SessionStateTimedOut
)

// Session represents an active user session
type Session struct {
	ID           string
	UserID       string
	Role         string
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
	LastActivity time.Time
	ExpiresAt    time.Time
	State        SessionState
	Metadata     map[string]interface{}
}

// SessionConfig represents session configuration
type SessionConfig struct {
	SessionTimeout     time.Duration
	AbsoluteTimeout    time.Duration
	MaxSessionsPerUser int
	IdleCheckInterval  time.Duration
	TokenLength        int
}

// SessionManager manages user sessions
type SessionManager struct {
	config      *SessionConfig
	sessions    map[string]*Session
	userIndex   map[string]map[string]bool // userID -> sessionIDs
	sessionLock sync.RWMutex
	userLock    sync.RWMutex
	logger      *zap.Logger
}

// NewSessionManager creates a new session manager
func NewSessionManager(config *SessionConfig, logger *zap.Logger) *SessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}

	return &SessionManager{
		config:    config,
		sessions:  make(map[string]*Session),
		userIndex: make(map[string]map[string]bool),
		logger:    logger,
	}
}

// DefaultSessionConfig returns the default session configuration
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		SessionTimeout:     30 * time.Minute,
		AbsoluteTimeout:    24 * time.Hour,
		MaxSessionsPerUser: 5,
		IdleCheckInterval:  1 * time.Minute,
		TokenLength:        32,
	}
}

// CreateSession creates a new session for a user
func (m *SessionManager) CreateSession(ctx context.Context, userID string, role string, ipAddress string, userAgent string) (*Session, error) {
	// Generate secure session token
	token, err := m.generateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %v", err)
	}

	now := time.Now()

	session := &Session{
		ID:           token,
		UserID:       userID,
		Role:         role,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    now,
		LastActivity: now,
		ExpiresAt:    now.Add(m.config.SessionTimeout),
		State:        SessionStateActive,
		Metadata:     make(map[string]interface{}),
	}

	// Check max sessions per user
	m.userLock.Lock()
	userSessions := m.userIndex[userID]
	if userSessions == nil {
		userSessions = make(map[string]bool)
		m.userIndex[userID] = userSessions
	}

	if len(userSessions) >= m.config.MaxSessionsPerUser {
		// Remove oldest session for this user
		m.removeOldestSession(userID)
	}
	m.userIndex[userID][token] = true
	m.userLock.Unlock()

	// Store session
	m.sessionLock.Lock()
	m.sessions[token] = session
	m.sessionLock.Unlock()

	m.logger.Info("Created new session",
		zap.String("session_id", token),
		zap.String("user_id", userID),
		zap.String("role", role),
		zap.String("ip_address", ipAddress),
		zap.Duration("timeout", m.config.SessionTimeout))

	return session, nil
}

// GetSession retrieves a session by token
func (m *SessionManager) GetSession(token string) (*Session, error) {
	m.sessionLock.RLock()
	defer m.sessionLock.RUnlock()

	session, exists := m.sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session has expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// ValidateSession validates a session token
func (m *SessionManager) ValidateSession(token string) (bool, error) {
	session, err := m.GetSession(token)
	if err != nil {
		return false, err
	}

	if session.State != SessionStateActive {
		return false, fmt.Errorf("session is not active")
	}

	return true, nil
}

// UpdateActivity updates the last activity timestamp
func (m *SessionManager) UpdateActivity(token string) error {
	m.sessionLock.Lock()
	defer m.sessionLock.Unlock()

	session, exists := m.sessions[token]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.LastActivity = time.Now()
	session.ExpiresAt = time.Now().Add(m.config.SessionTimeout)

	return nil
}

// RevokeSession revokes a session
func (m *SessionManager) RevokeSession(token string) error {
	m.sessionLock.Lock()
	defer m.sessionLock.Unlock()

	session, exists := m.sessions[token]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.State = SessionStateRevoked

	// Remove from user index
	m.userLock.Lock()
	if userSessions := m.userIndex[session.UserID]; userSessions != nil {
		delete(userSessions, token)
	}
	m.userLock.Unlock()

	// Remove session
	delete(m.sessions, token)

	m.logger.Info("Revoked session",
		zap.String("session_id", token),
		zap.String("user_id", session.UserID))

	return nil
}

// RevokeAllUserSessions revokes all sessions for a user
func (m *SessionManager) RevokeAllUserSessions(userID string) error {
	m.userLock.Lock()
	defer m.userLock.Unlock()

	userSessions := m.userIndex[userID]
	if userSessions == nil {
		return nil // No sessions to revoke
	}

	m.sessionLock.Lock()
	defer m.sessionLock.Unlock()

	for token := range userSessions {
		if session, exists := m.sessions[token]; exists {
			session.State = SessionStateRevoked
		}
		delete(m.sessions, token)
	}

	// Clear user index
	delete(m.userIndex, userID)

	m.logger.Info("Revoked all sessions for user",
		zap.String("user_id", userID),
		zap.Int("sessions_revoked", len(userSessions)))

	return nil
}

// GetUserSessions returns all sessions for a user
func (m *SessionManager) GetUserSessions(userID string) ([]*Session, error) {
	m.userLock.RLock()
	defer m.userLock.RUnlock()

	userSessions := m.userIndex[userID]
	if userSessions == nil {
		return []*Session{}, nil
	}

	m.sessionLock.RLock()
	defer m.sessionLock.RUnlock()

	sessions := make([]*Session, 0, len(userSessions))
	for token := range userSessions {
		if session, exists := m.sessions[token]; exists {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// GetActiveSessions returns all active sessions
func (m *SessionManager) GetActiveSessions() ([]*Session, error) {
	m.sessionLock.RLock()
	defer m.sessionLock.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.State == SessionStateActive {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// GetSessionCount returns the total number of active sessions
func (m *SessionManager) GetSessionCount() int {
	m.sessionLock.RLock()
	defer m.sessionLock.RUnlock()

	count := 0
	for _, session := range m.sessions {
		if session.State == SessionStateActive {
			count++
		}
	}
	return count
}

// StartCleanupRoutine starts the session cleanup routine
func (m *SessionManager) StartCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(m.config.IdleCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanupExpiredSessions()
		}
	}
}

// cleanupExpiredSessions removes expired and timed-out sessions
func (m *SessionManager) cleanupExpiredSessions() {
	now := time.Now()

	m.sessionLock.Lock()
	defer m.sessionLock.Unlock()

	expiredTokens := []string{}

	for token, session := range m.sessions {
		if session.State != SessionStateActive {
			continue
		}

		// Check session timeout
		if now.After(session.ExpiresAt) {
			session.State = SessionStateExpired
			expiredTokens = append(expiredTokens, token)

			m.userLock.Lock()
			if userSessions := m.userIndex[session.UserID]; userSessions != nil {
				delete(userSessions, token)
			}
			m.userLock.Unlock()

			m.logger.Info("Session expired due to timeout",
				zap.String("session_id", token),
				zap.String("user_id", session.UserID))
			continue
		}

		// Check absolute timeout
		if now.Sub(session.CreatedAt) > m.config.AbsoluteTimeout {
			session.State = SessionStateTimedOut
			expiredTokens = append(expiredTokens, token)

			m.userLock.Lock()
			if userSessions := m.userIndex[session.UserID]; userSessions != nil {
				delete(userSessions, token)
			}
			m.userLock.Unlock()

			m.logger.Info("Session timed out due to absolute timeout",
				zap.String("session_id", token),
				zap.String("user_id", session.UserID))
		}
	}

	// Remove expired sessions
	for _, token := range expiredTokens {
		delete(m.sessions, token)
	}
}

// removeOldestSession removes the oldest session for a user
func (m *SessionManager) removeOldestSession(userID string) {
	m.sessionLock.RLock()
	userSessions := m.userIndex[userID]
	m.sessionLock.RUnlock()

	if userSessions == nil {
		return
	}

	var oldestToken string
	var oldestTime time.Time

	m.sessionLock.RLock()
	for token := range userSessions {
		if session, exists := m.sessions[token]; exists {
			if oldestToken == "" || session.CreatedAt.Before(oldestTime) {
				oldestToken = token
				oldestTime = session.CreatedAt
			}
		}
	}
	m.sessionLock.RUnlock()

	if oldestToken != "" {
		m.RevokeSession(oldestToken)
	}
}

// generateToken generates a secure random token
func (m *SessionManager) generateToken() (string, error) {
	bytes := make([]byte, m.config.TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
