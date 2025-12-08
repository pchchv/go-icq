package wire

import (
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

func marshal(t reflect.Type) error {
	if t == nil {
		return errMarshalFailureNilSNAC
	}
	return nil
}
