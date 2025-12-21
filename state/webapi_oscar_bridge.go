package state

import (
	"context"
	"fmt"
	"time"
)

// OSCARBridgeSession represents a bridge between WebAPI and OSCAR sessions.
type OSCARBridgeSession struct {
	WebSessionID  string    // WebAPI session identifier
	OSCARCookie   []byte    // OSCAR authentication cookie
	BOSHost       string    // BOS server hostname
	BOSPort       int       // BOS server port
	UseSSL        bool      // Whether to use SSL connection
	ScreenName    string    // Screen name associated with the session
	ClientName    string    // Client application name
	ClientVersion string    // Client application version
	CreatedAt     time.Time // Bridge creation timestamp
	LastAccessed  time.Time // Last access timestamp
}

// OSCARBridgeStore manages the persistence of OSCAR bridge sessions in the database.
// It provides methods to store, retrieve,
// and manage the mapping between WebAPI sessions and OSCAR authentication cookies.
type OSCARBridgeStore struct {
	store *SQLiteUserStore
}

// NewOSCARBridgeStore creates a new OSCAR bridge store instance.
func (s *SQLiteUserStore) NewOSCARBridgeStore() *OSCARBridgeStore {
	return &OSCARBridgeStore{store: s}
}

// SaveBridgeSession stores the mapping between WebAPI and OSCAR sessions.
func (s *OSCARBridgeStore) SaveBridgeSession(
	ctx context.Context,
	webSessionID string,
	oscarCookie []byte,
	bosHost string,
	bosPort int) error {
	var screenName string
	now := time.Now()
	query := `
		INSERT INTO oscar_bridge_sessions 
		(web_session_id, oscar_cookie, bos_host, bos_port, screen_name, created_at, last_accessed)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(web_session_id) DO UPDATE SET
			oscar_cookie = excluded.oscar_cookie,
			bos_host = excluded.bos_host,
			bos_port = excluded.bos_port,
			last_accessed = excluded.last_accessed
	`
	_, err := s.store.db.ExecContext(ctx, query, webSessionID, oscarCookie, bosHost, bosPort, screenName, now, now)
	if err != nil {
		return fmt.Errorf("failed to save bridge session: %w", err)
	}

	return nil
}

// DeleteBridgeSession removes a bridge session.
func (s *OSCARBridgeStore) DeleteBridgeSession(ctx context.Context, webSessionID string) error {
	query := `DELETE FROM oscar_bridge_sessions WHERE web_session_id = ?`
	result, err := s.store.db.ExecContext(ctx, query, webSessionID)
	if err != nil {
		return fmt.Errorf("failed to delete bridge session: %w", err)
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	} else if rowsAffected == 0 {
		return fmt.Errorf("bridge session not found")
	}

	return nil
}

// CleanupExpiredSessions removes bridge sessions that haven't been accessed recently.
func (s *OSCARBridgeStore) CleanupExpiredSessions(ctx context.Context, maxAge time.Duration) (int, error) {
	cutoff := time.Now().Add(-maxAge)
	query := `DELETE FROM oscar_bridge_sessions WHERE last_accessed < ?`
	result, err := s.store.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	} else {
		return int(rowsAffected), nil
	}
}
