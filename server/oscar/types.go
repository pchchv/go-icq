package oscar

import (
	"context"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

type AdminService interface {
	ConfirmRequest(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame) (wire.SNACMessage, error)
	InfoQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x07_0x02_AdminInfoQuery) (wire.SNACMessage, error)
	InfoChangeRequest(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x07_0x04_AdminInfoChangeRequest) (wire.SNACMessage, error)
}

type AuthService interface {
	BUCPChallenge(ctx context.Context, inBody wire.SNAC_0x17_0x06_BUCPChallengeRequest, newUUID func() uuid.UUID) (wire.SNACMessage, error)
	BUCPLogin(ctx context.Context, inBody wire.SNAC_0x17_0x02_BUCPLoginRequest, advertisedHost string) (wire.SNACMessage, error)
	CrackCookie(authCookie []byte) (state.ServerCookie, error)
	FLAPLogin(ctx context.Context, inFrame wire.FLAPSignonFrame, advertisedHost string) (wire.TLVRestBlock, error)
	KerberosLogin(ctx context.Context, inBody wire.SNAC_0x050C_0x0002_KerberosLoginRequest, advertisedHost string) (wire.SNACMessage, error)
	RegisterBOSSession(ctx context.Context, authCookie state.ServerCookie) (*state.SessionInstance, error)
	RegisterChatSession(ctx context.Context, authCookie state.ServerCookie) (*state.SessionInstance, error)
	RetrieveBOSSession(ctx context.Context, authCookie state.ServerCookie) (*state.SessionInstance, error)
	Signout(ctx context.Context, instance *state.SessionInstance)
	SignoutChat(ctx context.Context, instance *state.SessionInstance)
}

type BARTService interface {
	UpsertItem(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x10_0x02_BARTUploadQuery) (wire.SNACMessage, error)
	RetrieveItem(ctx context.Context, inFrame wire.SNACFrame, inBody wire.SNAC_0x10_0x04_BARTDownloadQuery) (wire.SNACMessage, error)
	RetrieveItemV2(ctx context.Context, inFrame wire.SNACFrame, inBody wire.SNAC_0x10_0x06_BARTDownload2Query) ([]wire.SNACMessage, error)
}

// BuddyListRegistry is the interface for keeping track of users with active buddy lists.
// Once registered, a user becomes visible to other users' buddy lists and vice versa.
type BuddyListRegistry interface {
	ClearBuddyListRegistry(ctx context.Context) error
	RegisterBuddyList(ctx context.Context, user state.IdentScreenName) error
	UnregisterBuddyList(ctx context.Context, user state.IdentScreenName) error
}

type BuddyService interface {
	RightsQuery(ctx context.Context, inFrame wire.SNACFrame) wire.SNACMessage
	AddBuddies(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x03_0x04_BuddyAddBuddies) error
	DelBuddies(_ context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x03_0x05_BuddyDelBuddies) error
	AddTempBuddies(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x03_0x0F_BuddyAddTempBuddies) error
	DelTempBuddies(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x03_0x10_BuddyDelTempBuddies) error
}

type ChatNavService interface {
	CreateRoom(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate) (wire.SNACMessage, error)
	ExchangeInfo(ctx context.Context, inFrame wire.SNACFrame, inBody wire.SNAC_0x0D_0x03_ChatNavRequestExchangeInfo) (wire.SNACMessage, error)
	RequestChatRights(ctx context.Context, inFrame wire.SNACFrame) wire.SNACMessage
	RequestRoomInfo(ctx context.Context, inFrame wire.SNACFrame, inBody wire.SNAC_0x0D_0x04_ChatNavRequestRoomInfo) (wire.SNACMessage, error)
}

type ChatService interface {
	ChannelMsgToHost(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x0E_0x05_ChatChannelMsgToHost) (*wire.SNACMessage, error)
}

// ChatSessionManager is the interface for closing chat sessions when a client disconnects.
type ChatSessionManager interface {
	RemoveUserFromAllChats(user state.IdentScreenName)
}

// DepartureNotifier is the interface for sending buddy departure notifications when a client disconnects.
type DepartureNotifier interface {
	BroadcastBuddyArrived(ctx context.Context, screenName state.IdentScreenName, userInfo wire.TLVUserInfo) error
	BroadcastBuddyDeparted(ctx context.Context, instance *state.SessionInstance) error
}
