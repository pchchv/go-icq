package oscar

import (
	"errors"

	"github.com/pchchv/go-icq/wire"
)

// ErrRouteNotFound is an error that indicates a failure to find a matching route for an OSCAR protocol request.
var ErrRouteNotFound = errors.New("route not found")

// ResponseWriter is the interface for sending a SNAC response to
// the client from the server handlers.
type ResponseWriter interface {
	SendSNAC(frame wire.SNACFrame, body any) error
}
