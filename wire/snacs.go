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

type ICQUserSearchRecord struct {
	UIN           uint32
	Age           uint16
	Email         string `oscar:"len_prefix=uint16,nullterm"`
	Gender        uint8
	Authorization uint8
	OnlineStatus  uint16
	FirstName     string `oscar:"len_prefix=uint16,nullterm"`
	LastName      string `oscar:"len_prefix=uint16,nullterm"`
	Nickname      string `oscar:"len_prefix=uint16,nullterm"`
}

type ICQMetadata struct {
	UIN     uint32
	Seq     uint16
	ReqType uint16
}

type ICQMetadataWithSubType struct {
	ICQMetadata
	Optional *struct {
		ReqSubType uint16
	} `oscar:"optional"`
}

type ICQ_0x0041_DBQueryOfflineMsgReply struct {
	ICQMetadata
	SenderUIN uint32
	Year      uint16
	Month     uint8
	Day       uint8
	Hour      uint8
	Minute    uint8
	MsgType   uint8
	Flags     uint8
	Message   string `oscar:"len_prefix=uint16,nullterm"`
}
