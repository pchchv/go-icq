package wire

type SNACError struct {
	TLVRestBlock
	Code uint16
}
