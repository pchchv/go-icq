package wire

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

var (
	errInvalidStructTag      = errors.New("invalid struct tag")
	errMarshalFailureNilSNAC = errors.New("attempting to marshal a nil SNAC")
)

type oscarTag struct {
	hasCountPrefix bool
	countPrefix    reflect.Kind
	hasLenPrefix   bool
	lenPrefix      reflect.Kind
	optional       bool
	nullTerminated bool
}

func parseOSCARTag(tag reflect.StructTag) (oscTag oscarTag, err error) {
	val, ok := tag.Lookup("oscar")
	if !ok {
		return
	}

	for _, kv := range strings.Split(val, ",") {
		kvSplit := strings.SplitN(kv, "=", 2)
		if len(kvSplit) == 2 {
			switch kvSplit[0] {
			case "len_prefix":
				oscTag.hasLenPrefix = true
				switch kvSplit[1] {
				case "uint8":
					oscTag.lenPrefix = reflect.Uint8
				case "uint16":
					oscTag.lenPrefix = reflect.Uint16
				default:
					return oscTag, fmt.Errorf("%w: unsupported type %s. allowed types: uint8, uint16",
						errInvalidStructTag, kvSplit[1])
				}
			case "count_prefix":
				oscTag.hasCountPrefix = true
				switch kvSplit[1] {
				case "uint8":
					oscTag.countPrefix = reflect.Uint8
				case "uint16":
					oscTag.countPrefix = reflect.Uint16
				default:
					return oscTag, fmt.Errorf("%w: unsupported type %s. allowed types: uint8, uint16",
						errInvalidStructTag, kvSplit[1])
				}
			}
		} else {
			switch kvSplit[0] {
			case "optional":
				oscTag.optional = true
			case "nullterm":
				oscTag.nullTerminated = true
			default:
				return oscTag, fmt.Errorf("%w: unsupported struct tag %s",
					errInvalidStructTag, kvSplit[0])
			}
		}
	}

	if oscTag.hasCountPrefix && oscTag.hasLenPrefix {
		err = fmt.Errorf("%w: struct elem has both len_prefix and count_prefix", errInvalidStructTag)
	}

	return oscTag, err
}

func marshalUnsignedInt(intType reflect.Kind, intVal int, w io.Writer, order binary.ByteOrder) error {
	switch intType {
	case reflect.Uint8:
		if err := binary.Write(w, order, uint8(intVal)); err != nil {
			return err
		}
	case reflect.Uint16:
		if err := binary.Write(w, order, uint16(intVal)); err != nil {
			return err
		}
	default:
		panic(fmt.Sprintf("unsupported type %s. allowed types: uint8, uint16", intType))
	}
	return nil
}

func marshalString(oscTag oscarTag, v reflect.Value, w io.Writer, order binary.ByteOrder) error {
	str := v.String()
	if oscTag.nullTerminated && str != "" {
		str = str + "\x00"
	}

	if oscTag.hasLenPrefix {
		if err := marshalUnsignedInt(oscTag.lenPrefix, len(str), w, order); err != nil {
			return err
		}
	}

	if str == "" {
		return nil
	}

	return binary.Write(w, order, []byte(str))
}

func marshalStruct(t reflect.Type, v reflect.Value, oscTag oscarTag, w io.Writer, order binary.ByteOrder) error {
	// marshal ICQ messages in little endian order
	if t.Name() == "ICQMessageReplyEnvelope" {
		order = binary.LittleEndian
	}

	marshalEachField := func(w io.Writer) error {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i)
			if field.Type.Kind() == reflect.Ptr {
				if i != t.NumField()-1 {
					return fmt.Errorf("pointer type found at non-final field %s", field.Name)
				}
				if field.Type.Elem().Kind() != reflect.Struct {
					return fmt.Errorf("field %s must point to a struct, got %v instead", field.Name,
						field.Type.Elem().Kind())
				}
			}
			if err := marshal(field.Type, value, field.Tag, w, order); err != nil {
				return err
			}
		}
		return nil
	}
	if oscTag.hasLenPrefix {
		buf := &bytes.Buffer{}
		if err := marshalEachField(buf); err != nil {
			return err
		}
		// write struct length
		if err := marshalUnsignedInt(oscTag.lenPrefix, buf.Len(), w, order); err != nil {
			return err
		}
		// write struct bytes
		if buf.Len() > 0 {
			_, err := w.Write(buf.Bytes())
			return err
		}
		return nil
	}
	return marshalEachField(w)
}

func marshalArray(t reflect.Type, v reflect.Value, w io.Writer, order binary.ByteOrder) error {
	if t.Elem().Kind() == reflect.Struct {
		for j := 0; j < v.Len(); j++ {
			if err := marshalStruct(t.Elem(), v.Index(j), oscarTag{}, w, order); err != nil {
				return fmt.Errorf("error marshalling %s: %w", t.Elem().Kind(), err)
			}
		}
	} else {
		if err := binary.Write(w, order, v.Interface()); err != nil {
			return fmt.Errorf("error marshalling %s: %w", t.Elem().Kind(), err)
		}
	}
	return nil
}

func marshalInterface(v reflect.Value, w io.Writer, tag oscarTag, order binary.ByteOrder) error {
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("interface underlying type must be a struct, got %v instead", elem.Kind())
	}
	return marshalStruct(elem.Type(), elem, tag, w, order)
}

// TODO: only write to temporary buffer if len_prefix is set
func marshalSlice(t reflect.Type, v reflect.Value, oscTag oscarTag, w io.Writer, order binary.ByteOrder) error {
	buf := &bytes.Buffer{}
	if t.Elem().Kind() == reflect.Struct {
		for j := 0; j < v.Len(); j++ {
			if err := marshalStruct(t.Elem(), v.Index(j), oscarTag{}, buf, order); err != nil {
				return err
			}
		}
	} else {
		if err := binary.Write(buf, order, v.Interface()); err != nil {
			return fmt.Errorf("error marshalling %s", t.Elem().Kind())
		}
	}

	if oscTag.hasLenPrefix {
		if err := marshalUnsignedInt(oscTag.lenPrefix, buf.Len(), w, order); err != nil {
			return err
		}
	} else if oscTag.hasCountPrefix {
		if err := marshalUnsignedInt(oscTag.countPrefix, v.Len(), w, order); err != nil {
			return err
		}
	}

	if buf.Len() > 0 {
		_, err := w.Write(buf.Bytes())
		return err
	}

	return nil
}

func marshal(t reflect.Type) error {
	if t == nil {
		return errMarshalFailureNilSNAC
	}
	return nil
}
