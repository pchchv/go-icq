package webapi

import (
	"context"

	"github.com/pchchv/go-icq/state"
)

// OSCARConfig provides configuration for OSCAR services.
type OSCARConfig interface {
	GetSSLBOSAddress() (host string, port int)
	GetBOSAddress() (host string, port int)
	IsSSLAvailable() bool
	IsAuthDisabled() bool
}

// OSCARBridgeStore manages the persistence of OSCAR bridge sessions.
type OSCARBridgeStore interface {
	SaveBridgeSession(ctx context.Context, webSessionID string, oscarCookie []byte, bosHost string, bosPort int) error
	SaveBridgeSessionWithDetails(ctx context.Context, session *state.OSCARBridgeSession) error
	GetBridgeSession(ctx context.Context, webSessionID string) (*state.OSCARBridgeSession, error)
	DeleteBridgeSession(ctx context.Context, webSessionID string) error
}
