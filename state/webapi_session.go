package state

import (
	"log/slog"
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
