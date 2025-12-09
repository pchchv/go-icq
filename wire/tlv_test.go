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

func TestTLVList_Append(t *testing.T) {
	want := TLVList{
		{
			Tag:   0,
			Value: []byte(`0`),
		},
		{
			Tag:   1,
			Value: []byte(`1`),
		},
		{
			Tag:   2,
			Value: []byte(`2`),
		},
	}

	have := TLVList{}
	have.Append(NewTLVBE(0, []byte(`0`)))
	have.Append(NewTLVBE(1, []byte(`1`)))
	have.Append(NewTLVBE(2, []byte(`2`)))

	assert.Equal(t, want, have)
}

func TestTLVList_AppendList(t *testing.T) {
	want := TLVList{
		{
			Tag:   0,
			Value: []byte(`0`),
		},
		{
			Tag:   1,
			Value: []byte(`1`),
		},
		{
			Tag:   2,
			Value: []byte(`2`),
		},
	}

	have := TLVList{}
	have.AppendList([]TLV{
		NewTLVBE(0, []byte(`0`)),
		NewTLVBE(1, []byte(`1`)),
		NewTLVBE(2, []byte(`2`)),
	})

	assert.Equal(t, want, have)
}

func TestTLVList_Replace(t *testing.T) {
	tests := []struct {
		name        string
		given       TLVList
		want        TLVList
		replacement TLV
	}{
		{
			name: "replace multiple TLVs",
			given: TLVList{
				NewTLVLE(0x01, []byte{0x01}),
				NewTLVLE(0x02, []byte{0x02}),
				NewTLVLE(0x01, []byte{0x03}),
				NewTLVLE(0x03, []byte{0x04}),
			},
			replacement: NewTLVLE(0x01, []byte{0xAA}),
			want: TLVList{
				NewTLVLE(0x01, []byte{0xAA}),
				NewTLVLE(0x02, []byte{0x02}),
				NewTLVLE(0x01, []byte{0xAA}),
				NewTLVLE(0x03, []byte{0x04}),
			},
		},
		{
			name: "no matching tags",
			given: TLVList{
				NewTLVLE(0x01, []byte{0x01}),
				NewTLVLE(0x02, []byte{0x02}),
				NewTLVLE(0x01, []byte{0x03}),
				NewTLVLE(0x03, []byte{0x04}),
			},
			replacement: NewTLVLE(0x07, []byte{0xAA}),
			want: TLVList{
				NewTLVLE(0x01, []byte{0x01}),
				NewTLVLE(0x02, []byte{0x02}),
				NewTLVLE(0x01, []byte{0x03}),
				NewTLVLE(0x03, []byte{0x04}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.given.Replace(tt.replacement)
			assert.Equal(t, tt.want, tt.given)
		})
	}
}
