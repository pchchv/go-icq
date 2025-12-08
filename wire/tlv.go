package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// TLV represents dynamically typed data in the OSCAR protocol.
// Each message consists of a tag (or key) and a blob value.
// TLVs are typically grouped together in arrays.
type TLV struct {
	Tag   uint16
	Value []byte `oscar:"len_prefix=uint16"`
}

func newTLV(tag uint16, val any, order binary.ByteOrder) TLV {
	t := TLV{
		Tag: tag,
	}
	if _, ok := val.([]byte); ok {
		t.Value = val.([]byte)
	} else {
		buf := &bytes.Buffer{}
		switch order {
		case binary.BigEndian:
			if err := MarshalBE(val, buf); err != nil {
				panic(fmt.Sprintf("unable to create TLV: %s", err.Error()))
			}
		case binary.LittleEndian:
			if err := MarshalLE(val, buf); err != nil {
				panic(fmt.Sprintf("unable to create TLV: %s", err.Error()))
			}
		}
		t.Value = buf.Bytes()
	}
	return t
}
