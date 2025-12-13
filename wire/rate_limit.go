package wire

// RateLimitClassID identifies a rate limit class.
type RateLimitClassID uint16

// RateClass defines the configuration for computing rate-limiting behavior
// using an exponential moving average over time.
//
// Each incoming event contributes a time delta (in ms), and the average inter-event
// time is calculated over a moving window of the most recent N events (`WindowSize`).
// The resulting average is compared against threshold levels to determine the
// current rate status (e.g., limited, alert, clear, or disconnect).
type RateClass struct {
	ID              RateLimitClassID // Unique identifier for this rate class.
	WindowSize      int32            // Number of samples used in the moving average calculation.
	ClearLevel      int32            // If rate-limited and average exceeds this, rate-limiting is lifted.
	AlertLevel      int32            // If average is below this, an alert state is triggered.
	LimitLevel      int32            // If average is below this, rate-limiting is triggered.
	DisconnectLevel int32            // If average is below this, the session should be disconnected.
	MaxLevel        int32            // Maximum allowed value for the moving average.
}
