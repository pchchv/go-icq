package oscar

import (
	"errors"

	"github.com/pchchv/go-icq/server/oscar/middleware"
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
