package state

const ChatRoomTypeUserCreated ChatRoomType = "userCreated"

// ChatRoomType represents the type of chat room.
type ChatRoomType string

// ChatSession represents a user's session in a chat room.
type ChatSession struct {
	ChatSID    string
	AIMSid     string
	RoomID     string
	ScreenName string
	InstanceID int
	JoinedAt   int64
	LeftAt     *int64
}
