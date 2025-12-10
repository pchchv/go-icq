package wire

const (
	BOS         uint16 = 0x0000
	OService    uint16 = 0x0001
	Locate      uint16 = 0x0002
	Buddy       uint16 = 0x0003
	ICBM        uint16 = 0x0004
	Advert      uint16 = 0x0005
	Invite      uint16 = 0x0006
	Admin       uint16 = 0x0007
	Popup       uint16 = 0x0008
	PermitDeny  uint16 = 0x0009
	UserLookup  uint16 = 0x000A
	Stats       uint16 = 0x000B
	Translate   uint16 = 0x000C
	ChatNav     uint16 = 0x000D
	Chat        uint16 = 0x000E
	ODir        uint16 = 0x000F
	BART        uint16 = 0x0010
	Feedbag     uint16 = 0x0013
	ICQ         uint16 = 0x0015
	BUCP        uint16 = 0x0017
	Alert       uint16 = 0x0018
	Plugin      uint16 = 0x0022
	UnnamedFG24 uint16 = 0x0024
	MDir        uint16 = 0x0025
	ARS         uint16 = 0x044A
	Kerberos    uint16 = 0x050C
)

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
