package wire

import (
	"bytes"
	"fmt"
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

type FLAPSignonFrame struct {
	TLVRestBlock
	FLAPVersion uint32
}

// FLAPFrameDisconnect is the last FLAP frame sent to a client before disconnection.
// It differs from FLAPFrame in that there is no payload length prefix at the end,
// which causes pre-multi-conn Windows AIM clients to
// improperly handle server disconnections,
// as when the regular FLAPFrame type is used.
type FLAPFrameDisconnect struct {
	StartMarker uint8
	FrameType   uint8
	Sequence    uint16
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

// SendSignonFrame sends a signon FLAP frame containing a list of
// TLVs to authenticate or initiate a session.
func (f *FlapClient) SendSignonFrame(tlvs []TLV) error {
	signonFrame := FLAPSignonFrame{
		FLAPVersion: 1,
	}

	if len(tlvs) > 0 {
		signonFrame.AppendList(tlvs)
	}

	buf := &bytes.Buffer{}
	if err := MarshalBE(signonFrame, buf); err != nil {
		return err
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	flap := FLAPFrame{
		StartMarker: 42,
		FrameType:   FLAPFrameSignon,
		Sequence:    uint16(f.sequence),
		Payload:     buf.Bytes(),
	}
	if err := MarshalBE(flap, f.w); err != nil {
		return err
	}

	f.sequence++
	return nil
}

// ReceiveSignonFrame receives a signon FLAP response message.
func (f *FlapClient) ReceiveSignonFrame() (FLAPSignonFrame, error) {
	flap := FLAPFrame{}
	if err := UnmarshalBE(&flap, f.r); err != nil {
		return FLAPSignonFrame{}, err
	}

	signonFrame := FLAPSignonFrame{}
	if err := UnmarshalBE(&signonFrame, bytes.NewBuffer(flap.Payload)); err != nil {
		return FLAPSignonFrame{}, err
	}

	return signonFrame, nil
}

func (f *FlapClient) SendDataFrame(payload []byte) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	flap := FLAPFrame{
		StartMarker: 42,
		FrameType:   FLAPFrameData,
		Sequence:    uint16(f.sequence),
		Payload:     payload,
	}
	if err := MarshalBE(flap, f.w); err != nil {
		return err
	}

	f.sequence++
	return nil
}

func (f *FlapClient) SendKeepAliveFrame() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	flap := FLAPFrame{
		StartMarker: 42,
		FrameType:   FLAPFrameKeepAlive,
		Sequence:    uint16(f.sequence),
	}
	if err := MarshalBE(flap, f.w); err != nil {
		return err
	}

	f.sequence++
	return nil
}

// ReceiveFLAP receives a FLAP frame and body.
// It only returns a body if the FLAP frame is a data frame.
func (f *FlapClient) ReceiveFLAP() (FLAPFrame, error) {
	flap := FLAPFrame{}
	err := UnmarshalBE(&flap, f.r)
	if err != nil {
		err = fmt.Errorf("unable to unmarshal FLAP frame: %w", err)
	}

	return flap, err
}

func (f *FlapClient) String() string {
	return ""
}

// OldSignoff sends a signoff FLAP frame for
// legacy clients that do not support multi-connection
// (Windows AIM 1.x–4.1).
//
// When these clients receive this frame,
// they display a "connection lost" message and close the session.
// Unlike normal FLAP frames, this variant omits the payload size field.
// If the size field were present, the client would hang
// without displaying any message upon server disconnection.
func (f *FlapClient) OldSignoff() error {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	flap := FLAPFrameDisconnect{
		StartMarker: 42,
		FrameType:   FLAPFrameSignoff,
		Sequence:    uint16(f.sequence),
	}
	return MarshalBE(flap, f.w)
}

// NewSignoff sends a signoff FLAP frame for multi-connection clients.
//
// The frame includes a TLV block with additional metadata such as error codes.
// Client behavior depends on the version:
//   - AIM 4.3–5.x: the client minimizes and enters a "signed off" state.
//   - AIM 6.x–7.x: the client closes and displays a disconnection error.
func (f *FlapClient) NewSignoff(tlvs TLVRestBlock) error {
	tlvBuf := &bytes.Buffer{}
	if err := MarshalBE(tlvs, tlvBuf); err != nil {
		return err
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	flap := FLAPFrame{
		StartMarker: 42,
		FrameType:   FLAPFrameSignoff,
		Sequence:    uint16(f.sequence),
		Payload:     tlvBuf.Bytes(),
	}

	if err := MarshalBE(flap, f.w); err != nil {
		return err
	}

	f.sequence++
	return nil
}
