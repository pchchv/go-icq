package http

import (
	"context"

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
