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
