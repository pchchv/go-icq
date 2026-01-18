package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pchchv/go-icq/state"
	"golang.org/x/time/rate"
)

// APIKeyValidator defines methods for validating Web API keys.
type APIKeyValidator interface {
	// GetAPIKeyByDevKey retrieves and validates an API key by its dev_key value.
	GetAPIKeyByDevKey(ctx context.Context, devKey string) (*state.WebAPIKey, error)
	// UpdateLastUsed updates the last_used timestamp for an API key.
	UpdateLastUsed(ctx context.Context, devKey string) error
}

// RateLimitInfo contains rate limit metadata for a request.
type RateLimitInfo struct {
	Limit     int   // Total requests allowed per window.
	Remaining int   // Requests remaining in current window.
	Reset     int64 // Unix timestamp when the window resets.
	Allowed   bool  // Whether the request is allowed.
}

// RateLimiter manages per-devID rate limiting for the Web API.
type RateLimiter struct {
	limiters   *cache.Cache
	mu         sync.RWMutex
	windowSize time.Duration // Rate limit window size (default: 1 minute)
}

// NewRateLimiter creates a new rate limiter with automatic cleanup.
func NewRateLimiter() *RateLimiter {
	// create cache with 5 minute expiration and 10 minute cleanup interval
	c := cache.New(5*time.Minute, 10*time.Minute)
	return &RateLimiter{
		limiters:   c,
		windowSize: time.Minute, // default 1 minute window
	}
}

// rateLimiterEntry tracks rate limiting data for a single devID.
type rateLimiterEntry struct {
	limiter    *rate.Limiter
	limit      int
	windowSize time.Duration
	lastReset  time.Time
}
