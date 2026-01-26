package handlers

import (
	"context"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// Buddy represents a buddy in the buddy list.
type Buddy struct {
	AimID     string `json:"aimId"`
	State     string `json:"state"`
	AwayMsg   string `json:"awayMsg,omitempty"`
	UserType  string `json:"userType"`
	StatusMsg string `json:"statusMsg,omitempty"`
}

// BuddyGroup represents a group of buddies.
type BuddyGroup struct {
	Name    string  `json:"name"`
	Buddies []Buddy `json:"buddies"`
}

// BuddyListService defines methods for buddy list operations.
type BuddyListService interface {
	GetBuddyList(ctx context.Context, screenName state.IdentScreenName) ([]BuddyGroup, error)
}

// BuddyListRegistry defines methods for buddy list management.
type BuddyListRegistry interface {
	RegisterBuddyList(ctx context.Context, screenName state.IdentScreenName) error
	UnregisterBuddyList(ctx context.Context, screenName state.IdentScreenName) error
}

// AuthService defines methods needed for authentication.
type AuthService interface {
	BUCPChallenge(ctx context.Context, bodyIn wire.SNAC_0x17_0x06_BUCPChallengeRequest, newUUID func() uuid.UUID) (wire.SNACMessage, error)
	BUCPLogin(ctx context.Context, bodyIn wire.SNAC_0x17_0x02_BUCPLoginRequest, newUserFn func(screenName state.DisplayScreenName) (state.User, error), advertisedHost string) (wire.SNACMessage, error)
	RegisterBOSSession(ctx context.Context, authCookie state.ServerCookie) (*state.SessionInstance, error)
}

// SessionManager defines methods for OSCAR session management.
type SessionManager interface {
	AddSession(ctx context.Context, screenName state.DisplayScreenName) (*state.SessionInstance, error)
	RemoveSession(instance *state.SessionInstance)
	RelayToScreenName(ctx context.Context, screenName state.IdentScreenName, msg wire.SNACMessage)
}

// EndSessionResponse represents the response for endSession endpoint.
type EndSessionResponse struct {
	Response struct {
		StatusCode int    `json:"statusCode"`
		StatusText string `json:"statusText"`
	} `json:"response"`
}
