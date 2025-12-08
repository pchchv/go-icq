package wire

// TLV represents dynamically typed data in the OSCAR protocol.
// Each message consists of a tag (or key) and a blob value.
// TLVs are typically grouped together in arrays.
type TLV struct {
	Tag   uint16
	Value []byte `oscar:"len_prefix=uint16"`
}
