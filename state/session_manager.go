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

// RetrieveSession finds a session with a matching sessionID.
// Returns nil if session is not found.
func (s *InMemorySessionManager) RetrieveSession(screenName IdentScreenName) *Session {
	s.mapMutex.RLock()
	defer s.mapMutex.RUnlock()

	if rec, ok := s.store[screenName]; ok {
		if rec.sess.SignonComplete() {
			return rec.sess
		}
		return nil
	}

	return nil
}

func (s *InMemorySessionManager) retrieveByScreenNames(screenNames []IdentScreenName) (ret []*Session) {
	s.mapMutex.RLock()
	defer s.mapMutex.RUnlock()

	for _, sn := range screenNames {
		for _, rec := range s.store {
			if rec.sess.SignonComplete() && sn == rec.sess.IdentScreenName() {
				ret = append(ret, rec.sess)
			}
		}
	}

	return ret
}
