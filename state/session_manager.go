package state

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/pchchv/go-icq/wire"
)

var errSessConflict = errors.New("session conflict: another session was created concurrently for this user")

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

// RelayToAll relays a message to all sessions in the session pool.
func (s *InMemorySessionManager) RelayToAll(ctx context.Context, msg wire.SNACMessage) {
	s.mapMutex.RLock()
	defer s.mapMutex.RUnlock()

	for _, rec := range s.store {
		if !rec.sess.SignonComplete() {
			continue
		}
		s.maybeRelayMessage(ctx, msg, rec.sess)
	}
}

// RelayToScreenName relays a message to a session with a matching screen name.
func (s *InMemorySessionManager) RelayToScreenName(ctx context.Context, screenName IdentScreenName, msg wire.SNACMessage) {
	if sess := s.RetrieveSession(screenName); sess == nil {
		s.logger.WarnContext(ctx, "can't send notification because user is not online", "recipient", screenName, "message", msg)
	} else {
		s.maybeRelayMessage(ctx, msg, sess)
	}
}

// RelayToScreenNames relays a message to sessions with matching screenNames.
func (s *InMemorySessionManager) RelayToScreenNames(ctx context.Context, screenNames []IdentScreenName, msg wire.SNACMessage) {
	for _, sess := range s.retrieveByScreenNames(screenNames) {
		s.maybeRelayMessage(ctx, msg, sess)
	}
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

func (s *InMemorySessionManager) AddSession(ctx context.Context, screenName DisplayScreenName) (*Session, error) {
	s.mapMutex.Lock()
	active := s.findRec(screenName.IdentScreenName())
	if active != nil {
		// there's an active session that needs to be removed
		// don't hold the lock while we wait
		s.mapMutex.Unlock()

		// signal to callers that this session has to go
		active.sess.Close()

		select {
		// wait for RemoveSession to be called
		case <-active.removed:
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for previous session to terminate: %w", ctx.Err())
		}

		// the session has been removed, let's try to replace it
		s.mapMutex.Lock()
	}

	defer s.mapMutex.Unlock()

	// make sure a concurrent call didn't already add a session
	if active != nil && s.findRec(screenName.IdentScreenName()) != nil {
		return nil, errSessConflict
	}

	sess := NewSession()
	sess.SetIdentScreenName(screenName.IdentScreenName())
	sess.SetDisplayScreenName(screenName)
	s.store[sess.IdentScreenName()] = &sessionSlot{
		sess:    sess,
		removed: make(chan bool),
	}

	return sess, nil
}

// RemoveSession takes a session out of the session pool.
func (s *InMemorySessionManager) RemoveSession(sess *Session) {
	s.mapMutex.Lock()
	defer s.mapMutex.Unlock()

	if rec, ok := s.store[sess.IdentScreenName()]; ok && rec.sess == sess {
		delete(s.store, sess.IdentScreenName())
		close(rec.removed)
	}
}

// AllSessions returns all sessions in the session pool.
func (s *InMemorySessionManager) AllSessions() (sessions []*Session) {
	s.mapMutex.RLock()
	defer s.mapMutex.RUnlock()

	for _, rec := range s.store {
		if !rec.sess.SignonComplete() {
			continue
		}
		sessions = append(sessions, rec.sess)
	}

	return
}

func (s *InMemorySessionManager) maybeRelayMessage(ctx context.Context, msg wire.SNACMessage, sess *Session) {
	switch sess.RelayMessage(msg) {
	case SessSendClosed:
		s.logger.WarnContext(ctx, "can't send notification because the user's session is closed", "recipient", sess.IdentScreenName(), "message", msg)
	case SessQueueFull:
		s.logger.WarnContext(ctx, "can't send notification because queue is full", "recipient", sess.IdentScreenName(), "message", msg)
		sess.Close()
	}
}

func (s *InMemorySessionManager) findRec(identScreenName IdentScreenName) *sessionSlot {
	for _, rec := range s.store {
		if identScreenName == rec.sess.IdentScreenName() {
			return rec
		}
	}
	return nil
}
