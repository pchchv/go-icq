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
