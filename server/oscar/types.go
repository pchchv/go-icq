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
