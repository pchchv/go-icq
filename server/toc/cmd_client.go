package toc

import (
	"sync"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// ChatRegistry manages the chat rooms that a user is
// connected to during a TOC session.
// It maintains mappings between chat room identifiers,
// metadata, and active chat sessions.
//
// This struct provides thread-safe operations for adding, retrieving,
// and managing chat room metadata and associated sessions.
type ChatRegistry struct {
	sessions map[int]*state.SessionInstance // tracks active chat sessions by chat room ID
	lookup   map[int]wire.ICBMRoomInfo      // maps chat room IDs to their metadata
	nextID   int                            // incremental identifier for newly added chat rooms
	m        sync.RWMutex                   // synchronization primitive for concurrent access
}

// NewChatRegistry creates a new ChatRegistry instances.
func NewChatRegistry() *ChatRegistry {
	return &ChatRegistry{
		lookup:   make(map[int]wire.ICBMRoomInfo),
		sessions: make(map[int]*state.SessionInstance),
		m:        sync.RWMutex{},
	}
}
