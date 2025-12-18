package state

import (
	"errors"
	"time"
)

var (
	ErrDupChatRoom      = errors.New("chat room already exists")
	ErrChatRoomNotFound = errors.New("chat room not found")
)

// ChatRoom represents of a chat room.
type ChatRoom struct {
	name       string
	creator    IdentScreenName
	exchange   uint16
	createTime time.Time
}

// NewChatRoom creates a new ChatRoom instance.
func NewChatRoom(name string, creator IdentScreenName, exchange uint16) ChatRoom {
	return ChatRoom{
		name:     name,
		creator:  creator,
		exchange: exchange,
	}
}

// Creator returns the screen name of the user who created the chat room.
func (c ChatRoom) Creator() IdentScreenName {
	return c.creator
}

// InstanceNumber returns which instance chatroom exists in. Overflow chat
// rooms do not exist yet, so all chats happen in the same instance.
func (c ChatRoom) InstanceNumber() uint16 {
	return 0
}
