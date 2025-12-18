package state

import (
	"time"

	"github.com/pchchv/go-icq/wire"
)

// RateClassState tracks the rate limiting state for a
// specific rate class within a user's session.
//
// It embeds the static wire.RateClass configuration and maintains dynamic,
// per-session state used to evaluate rate limits in real time.
type RateClassState struct {
	// static rate limit configuration for this class
	wire.RateClass
	// CurrentLevel is the current exponential moving average for this rate class.
	CurrentLevel int32
	// LastTime represents the last time a SNAC message was sent for this rate class.
	LastTime time.Time
	// CurrentStatus is the last recorded rate limit status for this rate class.
	CurrentStatus wire.RateLimitStatus
	// Subscribed indicates whether the user wants to
	// receive rate limit parameter updates for this rate class.
	Subscribed bool
	// LimitedNow indicates whether the user is currently rate limited for this rate class.
	// The user is blocked from sending SNACs in this rate class until the clear threshold is met.
	LimitedNow bool
}
