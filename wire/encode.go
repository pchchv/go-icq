package wire

import "reflect"

type oscarTag struct {
	hasCountPrefix bool
	countPrefix    reflect.Kind
	hasLenPrefix   bool
	lenPrefix      reflect.Kind
	optional       bool
	nullTerminated bool
}
