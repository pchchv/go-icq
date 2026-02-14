package oscar

import "github.com/pchchv/go-icq/wire"

// ResponseWriter is the interface for sending a SNAC response to
// the client from the server handlers.
type ResponseWriter interface {
	SendSNAC(frame wire.SNACFrame, body any) error
}
