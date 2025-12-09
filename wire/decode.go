package wire

import (
	"bytes"
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

func unmarshalStruct(t reflect.Type, v reflect.Value, oscTag oscarTag, r io.Reader, order binary.ByteOrder) error {
	if oscTag.hasLenPrefix {
		bufLen, err := unmarshalUnsignedInt(oscTag.lenPrefix, r, order)
		if err != nil {
			return err
		}

		b := make([]byte, bufLen)
		if bufLen > 0 {
			if _, err := io.ReadFull(r, b); err != nil {
				return err
			}
		}

		r = bytes.NewBuffer(b)
	}

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if field.Type.Kind() == reflect.Ptr {
			if i != v.NumField()-1 {
				return fmt.Errorf("pointer type found at non-final field %s", field.Name)
			}
			if field.Type.Elem().Kind() != reflect.Struct {
				return fmt.Errorf("%w: field %s must point to a struct, got %v instead",
					errNonOptionalPointer, field.Name, field.Type.Elem().Kind())
			}
		}

		if err := unmarshal(field.Type, value, field.Tag, r, order); err != nil {
			return err
		}
	}

	return nil
}

func unmarshalArray(v reflect.Value, r io.Reader, order binary.ByteOrder) error {
	arrLen := v.Len()
	arrType := v.Type().Elem()
	for i := 0; i < arrLen; i++ {
		elem := reflect.New(arrType).Elem()
		if err := unmarshal(arrType, elem, "", r, order); err != nil {
			return err
		}
		v.Index(i).Set(elem)
	}

	return nil
}
