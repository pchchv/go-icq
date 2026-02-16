package oscar

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/pchchv/go-icq/config"
	"github.com/pchchv/go-icq/server/oscar/middleware"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

var (
	// ErrRouteNotFound is an error that indicates a failure to find a matching route for an OSCAR protocol request.
	ErrRouteNotFound            = errors.New("route not found")
	errUnknownICQMetaReqType    = errors.New("unknown ICQ request type")
	errUnknownICQMetaReqSubType = errors.New("unknown ICQ metadata request subtype")
)

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

func (h Handler) ICQDBQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x15_0x02_BQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	md, ok := inBody.Bytes(wire.ICQTLVTagsMetadata)
	if !ok {
		return errors.New("invalid ICQ frame")
	}

	icqChunk := wire.ICQMessageRequestEnvelope{}
	if err := wire.UnmarshalLE(&icqChunk, bytes.NewBuffer(md)); err != nil {
		return err
	}

	buf := bytes.NewBuffer(icqChunk.Body)
	icqMD := wire.ICQMetadataWithSubType{}
	if err := wire.UnmarshalLE(&icqMD, buf); err != nil {
		return err
	}

	switch icqMD.ReqType {
	case wire.ICQDBQueryOfflineMsgReq:
		return h.ICQService.OfflineMsgReq(ctx, instance, icqMD.Seq)
	case wire.ICQDBQueryDeleteMsgReq:
		return h.ICQService.DeleteMsgReq(ctx, instance, icqMD.Seq)
	case wire.ICQDBQueryMetaReq:
		if icqMD.Optional == nil {
			return errors.New("got req without subtype")
		}

		h.Logger.Debug(
			"ICQ client request",
			"query_name",
			wire.ICQDBQueryName(icqMD.ReqType),
			"query_type",
			wire.ICQDBQueryMetaName(icqMD.Optional.ReqSubType),
			"uin",
			instance.UIN(),
		)
		switch icqMD.Optional.ReqSubType {
		case wire.ICQDBQueryMetaReqShortInfo:
			userInfo := wire.ICQ_0x07D0_0x04BA_DBQueryMetaReqShortInfo{}
			if err := binary.Read(buf, binary.LittleEndian, &userInfo); err != nil {
				return nil
			}
			return h.ICQService.ShortUserInfo(ctx, instance, userInfo, icqMD.Seq)
		case wire.ICQDBQueryMetaReqFullInfo, wire.ICQDBQueryMetaReqFullInfo2:
			userInfo := wire.ICQ_0x07D0_0x051F_DBQueryMetaReqSearchByUIN{}
			if err := binary.Read(buf, binary.LittleEndian, &userInfo); err != nil {
				return nil
			}
			return h.ICQService.FullUserInfo(ctx, instance, userInfo, icqMD.Seq)
		case wire.ICQDBQueryMetaReqXMLReq:
			req := wire.ICQ_0x07D0_0x0898_DBQueryMetaReqXMLReq{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.XMLReqData(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetPermissions:
			req := wire.ICQ_0x07D0_0x0424_DBQueryMetaReqSetPermissions{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetPermissions(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSearchByUIN:
			req := wire.ICQ_0x07D0_0x051F_DBQueryMetaReqSearchByUIN{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.FindByUIN(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSearchByUIN2:
			rest := buf.Bytes()
			if bytes.HasPrefix(rest, []byte{0x36, 0x01, 0x06, 0x00}) && len(rest) == 8 {
				// fix incorrect TLV len set by QIP 2005. it specifies len=6
				// for a 4-byte value, causing the unmarshaler to return EOF.
				rest[2] = 4
			}

			req := wire.ICQ_0x07D0_0x0569_DBQueryMetaReqSearchByUIN2{}
			if err := wire.UnmarshalLE(&req, bytes.NewReader(rest)); err != nil {
				return err
			}

			if err := h.ICQService.FindByUIN2(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSearchByEmail:
			req := wire.ICQ_0x07D0_0x0529_DBQueryMetaReqSearchByEmail{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.FindByICQEmail(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSearchByEmail3:
			req := wire.ICQ_0x07D0_0x0573_DBQueryMetaReqSearchByEmail3{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.FindByEmail3(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSearchByDetails:
			req := wire.ICQ_0x07D0_0x0515_DBQueryMetaReqSearchByDetails{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.FindByICQName(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSearchWhitePages:
			req := wire.ICQ_0x07D0_0x0533_DBQueryMetaReqSearchWhitePages{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.FindByICQInterests(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSearchWhitePages2:
			req := wire.ICQ_0x07D0_0x055F_DBQueryMetaReqSearchWhitePages2{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.FindByWhitePages2(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetBasicInfo:
			req := wire.ICQ_0x07D0_0x03EA_DBQueryMetaReqSetBasicInfo{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetBasicInfo(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetWorkInfo:
			req := wire.ICQ_0x07D0_0x03F3_DBQueryMetaReqSetWorkInfo{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetWorkInfo(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetMoreInfo:
			req := wire.ICQ_0x07D0_0x03FD_DBQueryMetaReqSetMoreInfo{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetMoreInfo(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetNotes:
			req := wire.ICQ_0x07D0_0x0406_DBQueryMetaReqSetNotes{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetUserNotes(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetEmails:
			req := wire.ICQ_0x07D0_0x040B_DBQueryMetaReqSetEmails{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetEmails(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetInterests:
			req := wire.ICQ_0x07D0_0x0410_DBQueryMetaReqSetInterests{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetInterests(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqSetAffiliations:
			req := wire.ICQ_0x07D0_0x041A_DBQueryMetaReqSetAffiliations{}
			if err := wire.UnmarshalLE(&req, buf); err != nil {
				return err
			}

			if err := h.ICQService.SetAffiliations(ctx, instance, req, icqMD.Seq); err != nil {
				return err
			}
		case wire.ICQDBQueryMetaReqStat0a8c,
			wire.ICQDBQueryMetaReqStat0a96,
			wire.ICQDBQueryMetaReqStat0aaa,
			wire.ICQDBQueryMetaReqStat0ab4,
			wire.ICQDBQueryMetaReqStat0ab9,
			wire.ICQDBQueryMetaReqStat0abe,
			wire.ICQDBQueryMetaReqStat0ac8,
			wire.ICQDBQueryMetaReqStat0acd,
			wire.ICQDBQueryMetaReqStat0ad2,
			wire.ICQDBQueryMetaReqStat0ad7,
			wire.ICQDBQueryMetaReqStat0758:
			h.Logger.Debug("got a request for stats, not doing anything right now")
		case wire.ICQDBQueryMetaReqDirectoryQuery, wire.ICQDBQueryMetaReqDirectoryUpdate:
			h.Logger.Debug("got a directory query/update request, not implemented yet")
		default:
			return fmt.Errorf("%w: %X", errUnknownICQMetaReqSubType, icqMD.Optional.ReqSubType)
		}
	default:
		return fmt.Errorf("%w: %X", errUnknownICQMetaReqType, icqMD.ReqType)
	}

	return nil
}

func (h Handler) InviteRequest(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	h.LogRequest(ctx, inFrame, nil)
	return nil
}

func (h Handler) LocateGetDirInfo(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x02_0x0B_LocateGetDirInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.LocateService.DirInfo(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) LocateRightsQuery(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := h.LocateService.RightsQuery(ctx, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) LocateSetDirInfo(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x02_0x09_LocateSetDirInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.LocateService.SetDirInfo(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) LocateSetInfo(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x02_0x04_LocateSetInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return h.LocateService.SetInfo(ctx, instance, inBody)
}

func (h Handler) LocateSetKeywordInfo(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x02_0x0F_LocateSetKeywordInfo{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.LocateService.SetKeywordInfo(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) LocateUserInfoQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x02_0x05_LocateUserInfoQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.LocateService.UserInfoQuery(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) LocateUserInfoQuery2(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x02_0x15_LocateUserInfoQuery2{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	// SNAC functionality for LocateUserInfoQuery and LocateUserInfoQuery2 is
	// identical except for the Type field data type (uint16 vs uint32).
	wrappedBody := wire.SNAC_0x02_0x05_LocateUserInfoQuery{
		Type:       uint16(inBody.Type2),
		ScreenName: inBody.ScreenName,
	}
	outSNAC, err := h.LocateService.UserInfoQuery(ctx, instance, inFrame, wrappedBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) MDirRequest(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	h.LogRequest(ctx, inFrame, nil)
	return nil
}

func (h Handler) ODirInfoQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0F_0x02_InfoQuery{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.ODirService.InfoQuery(ctx, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, outSNAC, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) ODirKeywordListQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	outSNAC, err := h.ODirService.KeywordListQuery(ctx, inFrame)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) OServiceClientOnline(ctx context.Context, service uint16, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x01_0x02_OServiceClientOnline{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.Logger.InfoContext(ctx, "user signed on")
	h.LogRequest(ctx, inFrame, inBody)
	return h.OServiceService.ClientOnline(ctx, service, inBody, instance)
}

func (h Handler) OServiceClientVersions(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x01_0x17_OServiceClientVersions{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNACs := h.OServiceService.ClientVersions(ctx, instance, inFrame, inBody)
	for _, snac := range outSNACs {
		h.LogRequestAndResponse(ctx, inFrame, inBody, snac.Frame, snac.Body)
		if err := rw.SendSNAC(snac.Frame, snac.Body); err != nil {
			return err
		}
	}

	return nil
}

func (h Handler) OServiceIdleNotification(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x01_0x11_OServiceIdleNotification{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return h.OServiceService.IdleNotification(ctx, instance, inBody)
}

func (h Handler) OServiceNoop(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	// no-op keep-alive
	h.LogRequest(ctx, inFrame, nil)
	return nil
}

func (h Handler) OServiceProbeReq(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := h.OServiceService.ProbeReq(ctx, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) OServiceRateParamsQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := h.OServiceService.RateParamsQuery(ctx, instance, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) OServiceRateParamsSubAdd(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x01_0x08_OServiceRateParamsSubAdd{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.OServiceService.RateParamsSubAdd(ctx, instance, inBody)
	h.LogRequest(ctx, inFrame, inBody)
	return nil
}

func (h Handler) OServiceServiceRequest(ctx context.Context, service uint16, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter, listener config.Listener) error {
	inBody := wire.SNAC_0x01_0x04_OServiceServiceRequest{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.OServiceService.ServiceRequest(ctx, service, instance, inFrame, inBody, listener)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) OServiceSetUserInfoFields(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x01_0x1E_OServiceSetUserInfoFields{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC, err := h.OServiceService.SetUserInfoFields(ctx, instance, inFrame, inBody)
	if err != nil {
		return err
	}

	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) OServiceSetPrivacyFlags(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, _ ResponseWriter) error {
	inBody := wire.SNAC_0x01_0x14_OServiceSetPrivacyFlags{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.OServiceService.SetPrivacyFlags(ctx, inBody)
	h.LogRequest(ctx, inFrame, inBody)
	return nil
}

func (h Handler) OServiceUserInfoQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := h.OServiceService.UserInfoQuery(ctx, instance, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

func (h Handler) PermitDenyAddDenyListEntries(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x09_0x07_PermitDenyAddDenyListEntries{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return h.PermitDenyService.AddDenyListEntries(ctx, instance, inBody)
}

func (h Handler) PermitDenyAddPermListEntries(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x09_0x05_PermitDenyAddPermListEntries{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return h.PermitDenyService.AddPermListEntries(ctx, instance, inBody)
}

func (h Handler) PermitDenyDelDenyListEntries(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x09_0x08_PermitDenyDelDenyListEntries{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return h.PermitDenyService.DelDenyListEntries(ctx, instance, inBody)
}

func (h Handler) PermitDenyDelPermListEntries(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x09_0x06_PermitDenyDelPermListEntries{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	h.LogRequest(ctx, inFrame, inBody)
	return h.PermitDenyService.DelPermListEntries(ctx, instance, inBody)
}

func (h Handler) PermitDenyRightsQuery(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, rw ResponseWriter) error {
	outSNAC := h.PermitDenyService.RightsQuery(ctx, inFrame)
	h.LogRequestAndResponse(ctx, inFrame, nil, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}

// PermitDenySetGroupPermitMask sets the classes of users I can interact with.
// We don't apply any of these settings to the privacy mechanism,
// so just log them for now.
func (h Handler) PermitDenySetGroupPermitMask(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x09_0x04_PermitDenySetGroupPermitMask{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	var flags []string
	if inBody.IsFlagSet(wire.OServiceUserFlagUnconfirmed) {
		flags = append(flags, "wire.OServiceUserFlagUnconfirmed")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagAdministrator) {
		flags = append(flags, "wire.OServiceUserFlagAdministrator")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagAOL) {
		flags = append(flags, "wire.OServiceUserFlagAOL")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagOSCARPay) {
		flags = append(flags, "wire.OServiceUserFlagOSCARPay")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagOSCARFree) {
		flags = append(flags, "wire.OServiceUserFlagOSCARFree")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagUnavailable) {
		flags = append(flags, "wire.OServiceUserFlagUnavailable")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagICQ) {
		flags = append(flags, "wire.OServiceUserFlagICQ")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagWireless) {
		flags = append(flags, "wire.OServiceUserFlagWireless")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagInternal) {
		flags = append(flags, "wire.OServiceUserFlagInternal")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagFish) {
		flags = append(flags, "wire.OServiceUserFlagFish")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagBot) {
		flags = append(flags, "wire.OServiceUserFlagBot")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagBeast) {
		flags = append(flags, "wire.OServiceUserFlagBeast")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagOneWayWireless) {
		flags = append(flags, "wire.OServiceUserFlagOneWayWireless")
	}

	if inBody.IsFlagSet(wire.OServiceUserFlagOfficial) {
		flags = append(flags, "wire.OServiceUserFlagOfficial")
	}

	h.Logger.Info("set pd group mask", "flags", flags)
	h.LogRequest(ctx, inFrame, inBody)
	return nil
}

func (h Handler) PluginRequest(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, _ io.Reader, _ ResponseWriter) error {
	h.LogRequest(ctx, inFrame, nil)
	return nil
}

func (h Handler) StatsReportEvents(ctx context.Context, _ *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter) error {
	inBody := wire.SNAC_0x0B_0x03_StatsReportEvents{}
	if err := wire.UnmarshalBE(&inBody, r); err != nil {
		return err
	}

	outSNAC := h.StatsService.ReportEvents(ctx, inFrame, inBody)
	h.LogRequestAndResponse(ctx, inFrame, inBody, outSNAC.Frame, outSNAC.Body)
	return rw.SendSNAC(outSNAC.Frame, outSNAC.Body)
}
