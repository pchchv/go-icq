package webapi

import (
	"context"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
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

type ChatService interface {
	ChannelMsgToHost(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x0E_0x05_ChatChannelMsgToHost) (*wire.SNACMessage, error)
}

type ChatNavService interface {
	CreateRoom(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate) (wire.SNACMessage, error)
	ExchangeInfo(ctx context.Context, inFrame wire.SNACFrame, inBody wire.SNAC_0x0D_0x03_ChatNavRequestExchangeInfo) (wire.SNACMessage, error)
	RequestChatRights(ctx context.Context, inFrame wire.SNACFrame) wire.SNACMessage
	RequestRoomInfo(ctx context.Context, inFrame wire.SNACFrame, inBody wire.SNAC_0x0D_0x04_ChatNavRequestRoomInfo) (wire.SNACMessage, error)
}

// CookieBaker defines methods for issuing and verifying AIM authentication tokens ("cookies").
// These tokens are used for authenticating client sessions with AIM services.
type CookieBaker interface {
	// Crack verifies and decodes a previously issued authentication token.
	// Returns the original payload if the token is valid.
	Crack(data []byte) ([]byte, error)
	// Issue creates a new authentication token from the given payload.
	// The resulting token can later be verified using Crack.
	Issue(data []byte) ([]byte, error)
}

type BuddyService interface {
	AddBuddies(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x03_0x04_BuddyAddBuddies) error
	BroadcastBuddyDeparted(ctx context.Context, instance *state.SessionInstance) error
	DelBuddies(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x03_0x05_BuddyDelBuddies) error
	RightsQuery(ctx context.Context, inFrame wire.SNACFrame) wire.SNACMessage
}

// BuddyListRegistry is the interface for keeping track of users with active buddy lists.
// Once registered, a user becomes visible to other users' buddy lists and vice versa.
type BuddyListRegistry interface {
	RegisterBuddyList(ctx context.Context, user state.IdentScreenName) error
	UnregisterBuddyList(ctx context.Context, user state.IdentScreenName) error
}

// FeedbagManager provides methods to manage buddy lists.
type FeedbagManager interface {
	RetrieveFeedbag(ctx context.Context, screenName state.IdentScreenName) ([]wire.FeedbagItem, error)
	InsertItem(ctx context.Context, screenName state.IdentScreenName, item wire.FeedbagItem) error
	UpdateItem(ctx context.Context, screenName state.IdentScreenName, item wire.FeedbagItem) error
	DeleteItem(ctx context.Context, screenName state.IdentScreenName, item wire.FeedbagItem) error
}
