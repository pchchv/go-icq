package state

import (
	"errors"
	"fmt"
	"net/url"
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

// Cookie returns the chat room unique identifier.
func (c ChatRoom) Cookie() string {
	// According to Pidgin, the chat cookie is a 3-part identifier.
	// The third segment is the chat name, which is shown explicitly in the Pidgin code.
	// We can assume that the first two parts were the exchange and instance number.
	// As of now, Pidgin is the only client that cares about the cookie format,
	// and it only cares about the chat name segment.
	return fmt.Sprintf("%d-%d-%s", c.exchange, c.InstanceNumber(), c.name)
}

// Exchange returns which exchange the chat room belongs to.
func (c ChatRoom) Exchange() uint16 {
	return c.exchange
}

// Name returns the chat room name.
func (c ChatRoom) Name() string {
	return c.name
}

// URL creates a URL that can be used to join a chat room.
func (c ChatRoom) URL() *url.URL {
	// macOS client v4.0.9 requires the `roomname` param to precede `exchange` param.
	// Create the path using string concatenation rather than
	// url.Values because url.Values sorts the params alphabetically.
	opaque := fmt.Sprintf("gochat?roomname=%s&exchange=%d", url.QueryEscape(c.name), c.exchange)
	return &url.URL{
		Scheme: "aim",
		Opaque: opaque,
	}
}
