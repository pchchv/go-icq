package toc

import (
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

// IPRateLimiter provides per-IP rate limiting using a token bucket algorithm.
// It caches individual rate limiters per IP address with automatic TTL expiration.
type IPRateLimiter struct {
	cache *cache.Cache // In-memory cache of rate limiters keyed by IP
	rate  rate.Limit   // Allowed request rate (events per second)
	burst int          // Maximum burst size
}

// NewIPRateLimiter returns a new IPRateLimiter that limits each IP
// to the specified rate and burst,
// with limiter state expiring after the given TTL.
// Entries are retained for up to 2×TTL to reduce churn under frequent lookups.
func NewIPRateLimiter(rate rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		cache: cache.New(ttl, 2*ttl),
		rate:  rate,
		burst: burst,
	}
}
