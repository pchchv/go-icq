package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pchchv/go-icq/state"
	"golang.org/x/time/rate"
)

const (
	// ContextKeyAPIKey is the context key for storing the validated API key.
	ContextKeyAPIKey contextKey = "api_key"
	// ContextKeyDevID is the context key for storing the developer ID.
	ContextKeyDevID contextKey = "dev_id"
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

// Allow checks if a request from the given devID is allowed based on rate limits.
func (r *RateLimiter) Allow(devID string, limit int) bool {
	info := r.CheckRateLimit(devID, limit)
	return info.Allowed
}

// CheckRateLimit checks if a
// request from the given devID is allowed and returns rate limit info.
func (r *RateLimiter) CheckRateLimit(devID string, limit int) RateLimitInfo {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	// get or create limiter entry for this devID
	var entry *rateLimiterEntry
	if val, found := r.limiters.Get(devID); found {
		entry = val.(*rateLimiterEntry)
		// check if limit has changed
		if entry.limit != limit {
			// recreate limiter with new limit
			entry.limiter = rate.NewLimiter(rate.Every(r.windowSize/time.Duration(limit)), limit)
			entry.limit = limit
		}
	} else {
		// create new limiter with burst equal to limit (allows initial burst)
		entry = &rateLimiterEntry{
			limiter:    rate.NewLimiter(rate.Every(r.windowSize/time.Duration(limit)), limit),
			limit:      limit,
			windowSize: r.windowSize,
			lastReset:  now,
		}
		r.limiters.Set(devID, entry, cache.DefaultExpiration)
	}

	// check if request is allowed
	allowed := entry.limiter.Allow()
	// calculate remaining requests (approximate based on tokens available)
	tokens := entry.limiter.Tokens()
	remaining := int(tokens)
	if remaining < 0 {
		remaining = 0
	}

	// calculate reset time (next window start)
	resetTime := now.Add(r.windowSize).Unix()
	return RateLimitInfo{
		Limit:     limit,
		Remaining: remaining,
		Reset:     resetTime,
		Allowed:   allowed,
	}
}

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// rateLimiterEntry tracks rate limiting data for a single devID.
type rateLimiterEntry struct {
	limiter    *rate.Limiter
	limit      int
	windowSize time.Duration
	lastReset  time.Time
}
