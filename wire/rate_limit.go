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

// RateLimitClasses stores a fixed set of rate limit class definitions.
//
// Each RateClass defines thresholds and behavior for
// computing moving-average-based rate limits.
// This struct provides access to individual classes by ID or to the full set.
type RateLimitClasses struct {
	classes [5]RateClass // Indexed by class ID - 1
}

// NewRateLimitClasses creates a new RateLimitClasses instance from a
// fixed array of 5 RateClass definitions.
//
// Each RateClass must have a unique ID from 1 to 5,
// and the array is expected to be ordered such that
// classes[ID-1] corresponds to RateClass.ID == ID.
// No validation is performed on the input.
func NewRateLimitClasses(classes [5]RateClass) RateLimitClasses {
	return RateLimitClasses{
		classes: classes,
	}
}

// DefaultRateLimitClasses returns the default SNAC rate limit classes used at
// one point by the original AIM service,
// as memorialized by the iserverd project.
func DefaultRateLimitClasses() RateLimitClasses {
	return RateLimitClasses{
		classes: [5]RateClass{
			{
				ID:              1,
				WindowSize:      80,
				ClearLevel:      2500,
				AlertLevel:      2000,
				LimitLevel:      1500,
				DisconnectLevel: 800,
				MaxLevel:        6000,
			},
			{
				ID:              2,
				WindowSize:      80,
				ClearLevel:      3000,
				AlertLevel:      2000,
				LimitLevel:      1500,
				DisconnectLevel: 1000,
				MaxLevel:        6000,
			},
			{
				ID:              3,
				WindowSize:      20,
				ClearLevel:      5100,
				AlertLevel:      5000,
				LimitLevel:      4000,
				DisconnectLevel: 3000,
				MaxLevel:        6000,
			},
			{
				ID:              4,
				WindowSize:      20,
				ClearLevel:      5500,
				AlertLevel:      5300,
				LimitLevel:      4200,
				DisconnectLevel: 3000,
				MaxLevel:        8000,
			},
			{
				ID:              5,
				WindowSize:      10,
				ClearLevel:      5500,
				AlertLevel:      5300,
				LimitLevel:      4200,
				DisconnectLevel: 3000,
				MaxLevel:        8000,
			},
		},
	}
}

// Get returns the RateClass associated with the given class ID.
//
// The class ID must be between 1 and 5 inclusive.
// Calling Get with an invalid ID will panic.
func (r RateLimitClasses) Get(ID RateLimitClassID) RateClass {
	return r.classes[ID-1]
}

// All returns all defined RateClass entries in order of their class IDs.
func (r RateLimitClasses) All() [5]RateClass {
	return r.classes
}
