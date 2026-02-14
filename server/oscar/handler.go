package oscar

import (
	"context"
	"errors"
	"io"

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
