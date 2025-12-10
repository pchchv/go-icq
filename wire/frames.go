package wire

import (
	"io"
	"sync"
)

const (
	FLAPFrameData      uint8 = 0x02
	FLAPFrameError     uint8 = 0x03
	FLAPFrameSignon    uint8 = 0x01
	FLAPFrameSignoff   uint8 = 0x04
	FLAPFrameKeepAlive uint8 = 0x05
)

type SNACError struct {
	TLVRestBlock
	Code uint16
}

type FLAPFrame struct {
	StartMarker uint8
	FrameType   uint8
	Sequence    uint16
	Payload     []byte `oscar:"len_prefix=uint16"`
}

// FlapClient sends and receive FLAP frames to and from the server.
// It ensures that the message sequence numbers are
// properly incremented after sending each successive message.
// It is not safe to use with multiple goroutines without synchronization.
type FlapClient struct {
	sequence uint32
	r        io.Reader
	w        io.Writer
	mutex    sync.Mutex
}

// NewFlapClient creates a new FLAP client instance.
// startSeq is the initial sequence value, which is typically 0.
// r receives FLAP messages, w writes FLAP messages.
func NewFlapClient(startSeq uint32, r io.Reader, w io.Writer) *FlapClient {
	return &FlapClient{
		sequence: startSeq,
		r:        r,
		w:        w,
		mutex:    sync.Mutex{},
	}
}
