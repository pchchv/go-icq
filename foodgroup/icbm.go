package foodgroup

import (
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pchchv/go-icq/state"
)

// convoTracker keeps track of messages initiated from a sender to a recipient.
// A user (the warner) can only warn another user (the warnee)
// only if the warner has received a message from the warnee.
// The warner may only warn 1 time per message received from warnee.
// The warner may only warn the warnee up to 3 times per warn window.
type convoTracker struct {
	convos *cache.Cache
	warns  *cache.Cache
	window time.Duration
}

func newConvoTracker() *convoTracker {
	window := 1 * time.Hour
	return &convoTracker{
		convos: cache.New(window, window),
		warns:  cache.New(window, window),
		window: window,
	}
}

func (w *convoTracker) key(sender state.IdentScreenName, recip state.IdentScreenName) string {
	return sender.String() + recip.String()
}
