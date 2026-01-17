package state

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/pchchv/go-icq/server/webapi/types"
	"github.com/pchchv/go-icq/wire"
)

var (
	// ErrNoWebAPISession is returned when a WebAPI session is not found.
	ErrNoWebAPISession = errors.New("WebAPI session not found")
	// ErrWebAPISessionExpired is returned when a WebAPI session has expired.
	ErrWebAPISessionExpired = errors.New("WebAPI session expired")
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

// IsExpired checks if the session has expired.
func (s *WebAPISession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsSubscribedTo checks if the session is subscribed to a specific event type.
func (s *WebAPISession) IsSubscribedTo(eventType string) bool {
	for _, event := range s.Events {
		if event == eventType {
			return true
		}
	}
	return false
}

// handleBuddyArrived handles when a buddy comes online.
func (s *WebAPISession) handleBuddyArrived(msg wire.SNACMessage) {
	if !s.IsSubscribedTo("presence") {
		return
	}

	body, ok := msg.Body.(wire.SNAC_0x03_0x0B_BuddyArrived)
	if !ok {
		return
	}

	presenceEvent := types.PresenceEvent{
		AimID:    body.ScreenName,
		State:    "online",
		UserType: "aim",
	}

	s.EventQueue.Push(types.EventTypePresence, presenceEvent)
}

// handleBuddyDeparted handles when a buddy goes offline.
func (s *WebAPISession) handleBuddyDeparted(msg wire.SNACMessage) {
	if !s.IsSubscribedTo("presence") {
		return
	}

	body, ok := msg.Body.(wire.SNAC_0x03_0x0C_BuddyDeparted)
	if !ok {
		return
	}

	presenceEvent := types.PresenceEvent{
		AimID:    body.ScreenName,
		State:    "offline",
		UserType: "aim",
	}

	s.EventQueue.Push(types.EventTypePresence, presenceEvent)
}

// handleBuddyMessage handles buddy/presence SNAC messages.
func (s *WebAPISession) handleBuddyMessage(msg wire.SNACMessage) {
	switch msg.Frame.SubGroup {
	case wire.BuddyArrived:
		s.handleBuddyArrived(msg)
	case wire.BuddyDeparted:
		s.handleBuddyDeparted(msg)
	}
}

// handleIncomingIM handles incoming instant messages.
func (s *WebAPISession) handleIncomingIM(msg wire.SNACMessage) {
	if !s.IsSubscribedTo("im") {
		return
	}

	body, ok := msg.Body.(wire.SNAC_0x04_0x07_ICBMChannelMsgToClient)
	if !ok {
		return
	}

	// extract message text from TLV data
	var messageText string
	if msgData, hasMsg := body.TLVRestBlock.Bytes(wire.ICBMTLVAOLIMData); hasMsg {
		if text, err := wire.UnmarshalICBMMessageText(msgData); err == nil {
			messageText = text
		}
	}

	if messageText == "" {
		return
	}

	// check if it's an auto-response (channel 2)
	autoResponse := body.ChannelID == 0x0002

	// create IM event
	imEvent := types.IMEvent{
		From:      body.ScreenName,
		Message:   messageText,
		Timestamp: float64(time.Now().Unix()),
		AutoResp:  autoResponse,
	}

	s.EventQueue.Push(types.EventTypeIM, imEvent)
}

// handleTypingNotification handles typing notifications.
func (s *WebAPISession) handleTypingNotification(msg wire.SNACMessage) {
	if !s.IsSubscribedTo("typing") {
		return
	}

	body, ok := msg.Body.(wire.SNAC_0x04_0x14_ICBMClientEvent)
	if !ok {
		return
	}

	// event types:
	//   0=stopped typing
	//   1=text typed
	//   2=typing
	isTyping := body.Event == 1 || body.Event == 2
	typingEvent := types.TypingEvent{
		From:   body.ScreenName,
		Typing: isTyping,
	}

	s.EventQueue.Push(types.EventTypeTyping, typingEvent)
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

// GetSession retrieves a session by aimsid.
func (m *WebAPISessionManager) GetSession(ctx context.Context, aimsid string) (*WebAPISession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, exists := m.sessions[aimsid]; !exists {
		return nil, ErrNoWebAPISession
	} else if session.IsExpired() {
		return nil, ErrWebAPISessionExpired
	} else {
		return session, nil
	}
}

// GetSessionByUser retrieves a session by screen name.
func (m *WebAPISessionManager) GetSessionByUser(ctx context.Context, screenName IdentScreenName) (*WebAPISession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, exists := m.byUser[screenName]; !exists {
		return nil, ErrNoWebAPISession
	} else if session.IsExpired() {
		return nil, ErrWebAPISessionExpired
	} else {
		return session, nil
	}
}

// GetAllSessions returns all active sessions (for monitoring/admin).
func (m *WebAPISessionManager) GetAllSessions(ctx context.Context) []*WebAPISession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*WebAPISession, 0, len(m.sessions))
	for _, session := range m.sessions {
		if !session.IsExpired() {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// GetSessionsByScreenName returns all sessions for a given screen name.
func (m *WebAPISessionManager) GetSessionsByScreenName(ctx context.Context, screenName DisplayScreenName) []*WebAPISession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sessions []*WebAPISession
	identScreenName := screenName.IdentScreenName()

	// check both the byUser map and iterate through all sessions since a user might have multiple sessions
	for _, session := range m.sessions {
		if session.ScreenName.IdentScreenName() == identScreenName {
			sessions = append(sessions, session)
		}
	}

	return sessions
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
