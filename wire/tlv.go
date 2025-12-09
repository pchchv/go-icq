package wire

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// TLVBlock is a type of TLV array that has the TLV element count encoded as
// a 2-byte value at the beginning of the encoded blob.
type TLVBlock struct {
	TLVList `oscar:"count_prefix=uint16"`
}

// TLVLBlock is a type of TLV array that has the TLV blob byte-length encoded as
// a 2-byte value at the beginning of the encoded blob.
type TLVLBlock struct {
	TLVList `oscar:"len_prefix=uint16"`
}

// TLVRestBlock is a type of TLV array that does not have
// any length information encoded in the blob.
// This typically means that a given offset in the SNAC payload,
// the TLV occupies the "rest" of the payload.
type TLVRestBlock struct {
	TLVList
}

// TLV represents dynamically typed data in the OSCAR protocol.
// Each message consists of a tag (or key) and a blob value.
// TLVs are typically grouped together in arrays.
type TLV struct {
	Tag   uint16
	Value []byte `oscar:"len_prefix=uint16"`
}

// NewTLVBE creates a new TLV. Values are marshalled in big-endian order.
func NewTLVBE(tag uint16, val any) TLV {
	return newTLV(tag, val, binary.BigEndian)
}

// NewTLVLE creates a new TLV. Values are marshalled in little-endian order.
func NewTLVLE(tag uint16, val any) TLV {
	return newTLV(tag, val, binary.LittleEndian)
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

// TLVList is a list of TLV elements.
// It provides methods to append and access TLVs in the array.
// It provides methods that decode the data blob into the appropriate type at runtime.
// The caller assumes the TLV data type at runtime based on the protocol specification.
// These methods are not safe for read-write access by multiple goroutines.
type TLVList []TLV

// Append adds a TLV to the end of the TLV list.
func (s *TLVList) Append(tlv TLV) {
	*s = append(*s, tlv)
}

// AppendList adds a TLV list to the end of the TLV list.
func (s *TLVList) AppendList(tlvs []TLV) {
	*s = append(*s, tlvs...)
}

// Replace updates the values of TLVs in the list with the same tag as new.
// If no matching tag is found, the list remains unchanged.
func (s *TLVList) Replace(new TLV) {
	for i, old := range *s {
		if old.Tag == new.Tag {
			(*s)[i].Value = new.Value
		}
	}
}

// Uint8 retrieves a byte value from the TLVList associated with the specified tag.
//
// If the specified tag is found,
// the function returns the associated value as a uint8 and true.
// If the tag is not found, the function returns 0 and false.
func (s *TLVList) Uint8(tag uint16) (uint8, bool) {
	for _, tlv := range *s {
		if tag == tlv.Tag {
			if len(tlv.Value) > 0 {
				return tlv.Value[0], true
			}
		}
	}
	return 0, false
}
