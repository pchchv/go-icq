package state

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/pchchv/go-icq/server/webapi/types"
)

// WebAPISession represents an active Web AIM API session.
type WebAPISession struct {
	AimSID          string            // Unique session ID for web client
	ScreenName      DisplayScreenName // User identity
	OSCARSession    *Session          // Bridge to existing OSCAR session
	Events          []string          // Subscribed event types
	EventQueue      *types.EventQueue // Per-session event queue
	DevID           string            // Developer ID that created this session
	ClientName      string            // Client application name
	ClientVersion   string            // Client application version
	CreatedAt       time.Time         // Session creation time
	LastAccessed    time.Time         // Last activity time
	ExpiresAt       time.Time         // Session expiration time
	FetchTimeout    int               // Long-polling timeout in milliseconds
	TimeToNextFetch int               // Suggested delay before next fetch
	RemoteAddr      string            // Client IP address
	TempBuddies     map[string]bool   // Temporary buddies for this session only
	logger          *slog.Logger      // Logger for debugging
}

// WebAPISessionManager manages Web API sessions with thread-safe operations.
type WebAPISessionManager struct {
	sessions      map[string]*WebAPISession          // Keyed by aimsid
	byUser        map[IdentScreenName]*WebAPISession // Keyed by screen name
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewWebAPISessionManager creates a new WebAPI session manager.
func NewWebAPISessionManager() *WebAPISessionManager {
	mgr := &WebAPISessionManager{
		sessions:    make(map[string]*WebAPISession),
		byUser:      make(map[IdentScreenName]*WebAPISession),
		stopCleanup: make(chan struct{}),
	}

	// start cleanup goroutine to remove expired sessions
	mgr.cleanupTicker = time.NewTicker(1 * time.Minute)
	go mgr.cleanupExpiredSessions()

	return mgr
}

// Shutdown stops the session manager and cleans up resources.
func (m *WebAPISessionManager) Shutdown(ctx context.Context) {
	close(m.stopCleanup)

	m.mu.Lock()
	defer m.mu.Unlock()

	// close all event queues
	for _, session := range m.sessions {
		if session.EventQueue != nil {
			session.EventQueue.Close()
		}
	}

	// clear all sessions
	m.sessions = make(map[string]*WebAPISession)
	m.byUser = make(map[IdentScreenName]*WebAPISession)
}

// cleanupExpiredSessions periodically removes expired sessions.
func (m *WebAPISessionManager) cleanupExpiredSessions() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.mu.Lock()
			now := time.Now()
			var toRemove []string

			for aimsid, session := range m.sessions {
				if now.After(session.ExpiresAt) {
					toRemove = append(toRemove, aimsid)
				}
			}

			for _, aimsid := range toRemove {
				session := m.sessions[aimsid]
				delete(m.sessions, aimsid)
				delete(m.byUser, session.ScreenName.IdentScreenName())
				if session.EventQueue != nil {
					session.EventQueue.Close()
				}
			}
			m.mu.Unlock()
		case <-m.stopCleanup:
			m.cleanupTicker.Stop()
			return
		}
	}
}
