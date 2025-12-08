package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLVList_NewTLVBEPanic(t *testing.T) {
	// make sure NewTLVBE panics when it encounters an unsupported type,
	// in this case it's int.
	assert.Panics(t, func() {
		NewTLVBE(1, 30)
	})
}

func TestTLVList_NewTLVLEPanic(t *testing.T) {
	// make sure NewTLVLE panics when it encounters an unsupported type,
	// in this case it's int.
	assert.Panics(t, func() {
		NewTLVLE(1, 30)
	})
}
