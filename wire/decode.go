package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

var errNotNullTerminated = errors.New("nullterm tag is set, but string is not null-terminated")

func unmarshalUnsignedInt(intType reflect.Kind, r io.Reader, order binary.ByteOrder) (bufLen int, err error) {
	switch intType {
	case reflect.Uint8:
		var l uint8
		if err = binary.Read(r, order, &l); err != nil {
			return 0, err
		}
		bufLen = int(l)
	case reflect.Uint16:
		var l uint16
		if err = binary.Read(r, order, &l); err != nil {
			return 0, err
		}
		bufLen = int(l)
	default:
		panic(fmt.Sprintf("unsupported type %s. allowed types: uint8, uint16", intType))
	}
	return bufLen, nil
}

func unmarshalString(v reflect.Value, oscTag oscarTag, r io.Reader, order binary.ByteOrder) error {
	if !oscTag.hasLenPrefix {
		return fmt.Errorf("missing len_prefix tag")
	}

	bufLen, err := unmarshalUnsignedInt(oscTag.lenPrefix, r, order)
	if err != nil {
		return err
	}

	buf := make([]byte, bufLen)
	if bufLen > 0 {
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if oscTag.nullTerminated {
			if buf[len(buf)-1] != 0x00 {
				return errNotNullTerminated
			}
			buf = buf[0 : len(buf)-1] // remove null terminator
		}
	}

	v.SetString(string(buf))
	return nil
}
