package http

import (
	"context"
	"time"

	"github.com/pchchv/go-icq/state"
)

// BARTAssetManager defines methods for managing BART (Buddy ART) assets.
type BARTAssetManager interface {
	// BARTItem retrieves a BART asset by its hash.
	BARTItem(ctx context.Context, hash []byte) ([]byte, error)
	// InsertBARTItem inserts a BART asset.
	InsertBARTItem(ctx context.Context, hash []byte, blob []byte, itemType uint16) error
	// ListBARTItems returns BART assets filtered by type.
	ListBARTItems(ctx context.Context, itemType uint16) ([]state.BARTItem, error)
	// DeleteBARTItem deletes a BART asset by hash.
	DeleteBARTItem(ctx context.Context, hash []byte) error
}

// ChatSessionRetriever defines a method for
// retrieving all sessions associated with a specific chat room.
type ChatSessionRetriever interface {
	// AllSessions returns all active sessions in the chat room identified by cookie.
	AllSessions(cookie string) []*state.Session
}

// ChatRoomCreator defines a method for creating a new chat room.
type ChatRoomCreator interface {
	// CreateChatRoom creates a new chat room.
	CreateChatRoom(ctx context.Context, chatRoom *state.ChatRoom) error
}

// ChatRoomRetriever defines a method for retrieving all chat rooms under a specific exchange.
type ChatRoomRetriever interface {
	// AllChatRooms returns all chat rooms associated with the given exchange ID.
	AllChatRooms(ctx context.Context, exchange uint16) ([]state.ChatRoom, error)
}

// ChatRoomDeleter defines a method for deleting chat rooms.
type ChatRoomDeleter interface {
	// DeleteChatRooms deletes chat rooms by their names under a specific exchange.
	DeleteChatRooms(ctx context.Context, exchange uint16, names []string) error
}

// BuddyBroadcaster defines a method for broadcasting presence updates.
type BuddyBroadcaster interface {
	// BroadcastVisibility sends presence updates to the specified filter list.
	// If sendDepartures is true, departure events are sent as well.
	BroadcastVisibility(ctx context.Context, you *state.SessionInstance, filter []state.IdentScreenName, sendDepartures bool) error
}

type aimChatUserHandle struct {
	ID         string `json:"id"`
	ScreenName string `json:"screen_name"`
}

type chatRoom struct {
	URL          string              `json:"url"`
	Name         string              `json:"name"`
	CreatorID    string              `json:"creator_id,omitempty"`
	CreateTime   time.Time           `json:"create_time"`
	Participants []aimChatUserHandle `json:"participants"`
}

type chatRoomCreate struct {
	Name string `json:"name"`
}

type chatRoomDelete struct {
	Names []string `json:"names"`
}
