package wire

// ICQMessageReplyEnvelope is a helper struct that provides syntactic sugar for
// marshaling an ICQ message into a little-endian byte array.
type ICQMessageReplyEnvelope struct {
	Message any `oscar:"len_prefix=uint16"`
}

// ICQMessageRequestEnvelope is a helper struct that provides syntactic sugar for
// unmarshaling an ICQ message into a little-endian byte array.
type ICQMessageRequestEnvelope struct {
	Body []byte `oscar:"len_prefix=uint16"`
}
