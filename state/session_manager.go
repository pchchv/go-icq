package state

import (
	"log/slog"
	"sync"
)

type sessionSlot struct {
	sess    *Session
	removed chan bool
}

// InMemorySessionManager handles the lifecycle of a user session and
// provides synchronized message relay between sessions in the session pool.
// An InMemorySessionManager is safe for concurrent use by multiple goroutines.
type InMemorySessionManager struct {
	store    map[IdentScreenName]*sessionSlot
	mapMutex sync.RWMutex
	logger   *slog.Logger
}

// NewInMemorySessionManager creates a new instance of InMemorySessionManager.
func NewInMemorySessionManager(logger *slog.Logger) *InMemorySessionManager {
	return &InMemorySessionManager{
		logger: logger,
		store:  make(map[IdentScreenName]*sessionSlot),
	}
}
