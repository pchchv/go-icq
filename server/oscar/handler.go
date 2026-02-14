package oscar

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/pchchv/go-icq/server/oscar/middleware"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// ErrRouteNotFound is an error that indicates a failure to find a matching route for an OSCAR protocol request.
var ErrRouteNotFound = errors.New("route not found")

// ResponseWriter is the interface for sending a SNAC response to
// the client from the server handlers.
type ResponseWriter interface {
	SendSNAC(frame wire.SNACFrame, body any) error
}

// Handler defines a structure for routing OSCAR protocol requests to
// appropriate handlers based on group:subGroup identifiers.
type Handler struct {
	AdminService
	BARTService
	BuddyService
	ChatNavService
	ChatService
	FeedbagService
	ICBMService
	ICQService
	LocateService
	ODirService
	OServiceService
	PermitDenyService
	StatsService
	UserLookupService
	middleware.RouteLogger
}

func (rt Handler) AdminInfoQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x07_0x02_AdminInfoQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.AdminService.InfoQuery(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) AdminInfoChangeRequest(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x07_0x04_AdminInfoChangeRequest{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.AdminService.InfoChangeRequest(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) AdminConfirmRequest(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC, err := rt.AdminService.ConfirmRequest(ctx, instance, inFrame)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) AlertNotifyCapabilities(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	rt.LogRequest(ctx, inFrame, nil)
	return nil
}

func (rt Handler) AlertNotifyDisplayCapabilities(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	rt.LogRequest(ctx, inFrame, nil)
	return nil
}

func (rt Handler) BARTUploadQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x10_0x02_BARTUploadQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.BARTService.UpsertItem(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, outSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) BARTDownloadQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x10_0x04_BARTDownloadQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.BARTService.RetrieveItem(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, outSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) BARTDownload2Query(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x10_0x06_BARTDownload2Query{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNACS, err := rt.BARTService.RetrieveItemV2(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	for _, snac := range outSNACS {
		rt.LogRequestAndResponse(ctx, inFrame, snac, snac.Frame, snac.Body)
		if err := rw.SendSNAC(snac.Frame, snac.Body); err != nil {
			return err
		}
	}

	return nil
}

func (rt Handler) BuddyAddBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x04_BuddyAddBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	rt.LogRequest(ctx, inFrame, inSNAC)
	return rt.BuddyService.AddBuddies(ctx, instance, inSNAC)
}

func (rt Handler) BuddyAddTempBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x0F_BuddyAddTempBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	rt.LogRequest(ctx, inFrame, inSNAC)
	return rt.BuddyService.AddTempBuddies(ctx, instance, inSNAC)
}

func (rt Handler) BuddyDelBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x05_BuddyDelBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	rt.LogRequest(ctx, inFrame, inSNAC)
	return rt.BuddyService.DelBuddies(ctx, instance, inSNAC)
}

func (rt Handler) BuddyDelTempBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x10_BuddyDelTempBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	rt.LogRequest(ctx, inFrame, inSNAC)
	return rt.BuddyService.DelTempBuddies(ctx, instance, inSNAC)
}

func (rt Handler) BuddyRightsQuery(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x02_BuddyRightsQuery{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	outSNAC := rt.BuddyService.RightsQuery(ctx, inFrame)
	rt.LogRequestAndResponse(ctx, inFrame, inSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) ChatChannelMsgToHost(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0E_0x05_ChatChannelMsgToHost{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.ChatService.ChannelMsgToHost(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		return nil
	}

	rt.Logger.InfoContext(ctx, "user sent a chat message")
	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) ChatNavCreateRoom(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.ChatNavService.CreateRoom(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	roomName, _ := inBody.String(wire.ChatRoomTLVRoomName)
	rt.Logger.InfoContext(ctx, "user started a chat room", slog.String("roomName", roomName))
	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) ChatNavRequestChatRights(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := rt.ChatNavService.RequestChatRights(ctx, inFrame)
	rt.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) ChatNavRequestExchangeInfo(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0D_0x03_ChatNavRequestExchangeInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.ChatNavService.ExchangeInfo(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) ChatNavRequestRoomInfo(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0D_0x04_ChatNavRequestRoomInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.ChatNavService.RequestRoomInfo(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) FeedbagDeleteItem(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x0A_FeedbagDeleteItem{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.FeedbagService.DeleteItem(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		rt.LogRequest(ctx, inFrame, inBody)
		return nil
	}

	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) FeedbagEndCluster(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	rt.FeedbagService.EndCluster(ctx, instance, inFrame)
	rt.LogRequest(ctx, inFrame, nil)
	return nil
}

func (rt Handler) FeedbagInsertItem(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x08_FeedbagInsertItem{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.FeedbagService.UpsertItem(ctx, instance, inFrame, inBody.Items)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		rt.LogRequest(ctx, inFrame, inBody)
		return nil
	}

	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) FeedbagQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC, err := rt.FeedbagService.Query(ctx, instance, inFrame)
	if err != nil {
		return err
	}

	rt.LogRequest(ctx, inFrame, outSNAC)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) FeedbagQueryIfModified(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x05_FeedbagQueryIfModified{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.FeedbagService.QueryIfModified(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) FeedbagRespondAuthorizeToHost(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x1A_FeedbagRespondAuthorizeToHost{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	if err := rt.FeedbagService.RespondAuthorizeToHost(ctx, instance, inFrame, inBody); err != nil {
		return err
	}

	rt.LogRequest(ctx, inFrame, inBody)
	return nil
}

func (rt Handler) FeedbagRightsQuery(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x02_FeedbagRightsQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC := rt.FeedbagService.RightsQuery(ctx, inFrame)
	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) FeedbagStartCluster(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x11_FeedbagStartCluster{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	rt.FeedbagService.StartCluster(ctx, instance, inFrame, inBody)
	rt.LogRequest(ctx, inFrame, inBody)
	return nil
}

func (rt Handler) FeedbagUpdateItem(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x09_FeedbagUpdateItem{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := rt.FeedbagService.UpsertItem(ctx, instance, inFrame, inBody.Items)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		rt.LogRequest(ctx, inFrame, inBody)
		return nil
	}

	rt.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (rt Handler) FeedbagUse(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	rt.LogRequest(ctx, inFrame, nil)
	return rt.FeedbagService.Use(ctx, instance)
}
