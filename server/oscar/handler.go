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

func (h Handler) AdminInfoQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x07_0x02_AdminInfoQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.AdminService.InfoQuery(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) AdminInfoChangeRequest(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x07_0x04_AdminInfoChangeRequest{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.AdminService.InfoChangeRequest(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) AdminConfirmRequest(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC, err := h.AdminService.ConfirmRequest(ctx, instance, inFrame)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) AlertNotifyCapabilities(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	h.LogRequest(ctx, inFrame, nil)
	return nil
}

func (h Handler) AlertNotifyDisplayCapabilities(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	h.LogRequest(ctx, inFrame, nil)
	return nil
}

func (h Handler) BARTUploadQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x10_0x02_BARTUploadQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.BARTService.UpsertItem(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, outSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) BARTDownloadQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x10_0x04_BARTDownloadQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.BARTService.RetrieveItem(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, outSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) BARTDownload2Query(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x10_0x06_BARTDownload2Query{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNACS, err := h.BARTService.RetrieveItemV2(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	for _, snac := range outSNACS {
		h.LogRequestAndResponse(ctx, inFrame, snac, snac.Frame, snac.Body)
		if err := rw.SendSNAC(snac.Frame, snac.Body); err != nil {
			return err
		}
	}

	return nil
}

func (h Handler) BuddyAddBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x04_BuddyAddBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inSNAC)
	return h.BuddyService.AddBuddies(ctx, instance, inSNAC)
}

func (h Handler) BuddyAddTempBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x0F_BuddyAddTempBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inSNAC)
	return h.BuddyService.AddTempBuddies(ctx, instance, inSNAC)
}

func (h Handler) BuddyDelBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x05_BuddyDelBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inSNAC)
	return h.BuddyService.DelBuddies(ctx, instance, inSNAC)
}

func (h Handler) BuddyDelTempBuddies(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x10_BuddyDelTempBuddies{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inSNAC)
	return h.BuddyService.DelTempBuddies(ctx, instance, inSNAC)
}

func (h Handler) BuddyRightsQuery(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inSNAC := wire.SNAC_0x03_0x02_BuddyRightsQuery{}
	if err := wire.UnmarshalBE(&inSNAC, r); err != nil {
		return err
	}

	outSNAC := h.BuddyService.RightsQuery(ctx, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, inSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ChatChannelMsgToHost(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0E_0x05_ChatChannelMsgToHost{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.ChatService.ChannelMsgToHost(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		return nil
	}

	h.Logger.InfoContext(ctx, "user sent a chat message")
	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ChatNavCreateRoom(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.ChatNavService.CreateRoom(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	roomName, _ := inBody.String(wire.ChatRoomTLVRoomName)
	h.Logger.InfoContext(ctx, "user started a chat room", slog.String("roomName", roomName))
	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ChatNavRequestChatRights(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := h.ChatNavService.RequestChatRights(ctx, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ChatNavRequestExchangeInfo(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0D_0x03_ChatNavRequestExchangeInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.ChatNavService.ExchangeInfo(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ChatNavRequestRoomInfo(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0D_0x04_ChatNavRequestRoomInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.ChatNavService.RequestRoomInfo(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) FeedbagDeleteItem(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x0A_FeedbagDeleteItem{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.FeedbagService.DeleteItem(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		h.LogRequest(ctx, inFrame, inBody)
		return nil
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) FeedbagEndCluster(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	h.FeedbagService.EndCluster(ctx, instance, inFrame)
	h.LogRequest(ctx, inFrame, nil)
	return nil
}

func (h Handler) FeedbagInsertItem(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x08_FeedbagInsertItem{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.FeedbagService.UpsertItem(ctx, instance, inFrame, inBody.Items)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		h.LogRequest(ctx, inFrame, inBody)
		return nil
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) FeedbagQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC, err := h.FeedbagService.Query(ctx, instance, inFrame)
	if err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, outSNAC)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) FeedbagQueryIfModified(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x05_FeedbagQueryIfModified{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.FeedbagService.QueryIfModified(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) FeedbagRespondAuthorizeToHost(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x1A_FeedbagRespondAuthorizeToHost{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	if err := h.FeedbagService.RespondAuthorizeToHost(ctx, instance, inFrame, inBody); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return nil
}

func (h Handler) FeedbagRightsQuery(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x02_FeedbagRightsQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC := h.FeedbagService.RightsQuery(ctx, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) FeedbagStartCluster(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x11_FeedbagStartCluster{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.FeedbagService.StartCluster(ctx, instance, inFrame, inBody)
	h.LogRequest(ctx, inFrame, inBody)
	return nil
}

func (h Handler) FeedbagUpdateItem(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x13_0x09_FeedbagUpdateItem{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.FeedbagService.UpsertItem(ctx, instance, inFrame, inBody.Items)
	if err != nil {
		return err
	}

	if outSNAC == nil {
		h.LogRequest(ctx, inFrame, inBody)
		return nil
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) FeedbagUse(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	h.LogRequest(ctx, inFrame, nil)
	return h.FeedbagService.Use(ctx, instance)
}

func (h Handler) ICBMAddParameters(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x04_0x02_ICBMAddParameters{}
	h.LogRequest(ctx, inFrame, inBody)
	return wire.UnmarshalBE(&inBody, r)
}

func (h Handler) ICBMChannelMsgToHost(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x04_0x06_ICBMChannelMsgToHost{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.ICBMService.ChannelMsgToHost(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.Logger.InfoContext(ctx, "user sent an IM", slog.String("recipient", inBody.ScreenName))
	if outSNAC == nil {
		return nil
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ICBMClientErr(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x04_0x0B_ICBMClientErr{}
	h.LogRequest(ctx, inFrame, inBody)
	err := wire.UnmarshalBE(&inBody, r)
	if err != nil {
		return err
	}

	return h.ICBMService.ClientErr(ctx, instance, inFrame, inBody)
}

func (h Handler) ICBMClientEvent(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x04_0x14_ICBMClientEvent{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return h.ICBMService.ClientEvent(ctx, instance, inFrame, inBody)
}

func (h Handler) ICBMEvilRequest(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x04_0x08_ICBMEvilRequest{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.ICBMService.EvilRequest(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ICBMOfflineRetrieve(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, rw ResponseWriter) error {
	outSNAC, err := h.ICBMService.OfflineRetrieve(ctx, instance, inFrame)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ICBMParameterQuery(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := h.ICBMService.ParameterQuery(ctx, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, outSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}
