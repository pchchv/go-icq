package oscar

import (
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/time/rate"
)

// IPRateLimiter enforces a per-IP rate limit using a token bucket algorithm.
// It caches individual rate limiters by IP address and
// supports tagging requests as originating from the BUCP or FLAP auth.
//
// The limiter uses an in-memory cache with TTL expiration,
// so rate limits reset after the TTL if no activity is observed for a given IP.
type IPRateLimiter struct {
	cache *cache.Cache // in-memory cache mapping IPs to rate limiters with optional BUCP tag
	rate  rate.Limit   // requests allowed per second
	burst int          // maximum burst size allowed
}

// NewIPRateLimiter initializes a new IPRateLimiter with the specified rate,
// burst size, and TTL for each IP's limiter.
// Entries expire after 2×TTL.
func NewIPRateLimiter(rate rate.Limit, burst int, ttl time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		cache: cache.New(ttl, 2*ttl),
		rate:  rate,
		burst: burst,
	}
}

// Allow checks if a request from the given IP is allowed under its rate limit.
// It returns whether the request is allowed and
// whether the connection uses BUCP auth.
func (l *IPRateLimiter) Allow(ip string) (allowed bool, isBUCP bool) {
	limiter, found := l.cache.Get(ip)
	if !found {
		limiter = &rateLimitEntry{
			limiter: rate.NewLimiter(l.rate, l.burst),
		}
		l.cache.Set(ip, limiter, cache.DefaultExpiration)
	}
	
	entry := limiter.(*rateLimitEntry)
	return entry.limiter.Allow(), entry.isBUCP
}

// SetBUCP marks the rate limiter for the given IP as
// originating from BUCP auth (default FLAP auth).
func (l *IPRateLimiter) SetBUCP(ip string) {
	limiter, found := l.cache.Get(ip)
	if !found {
		limiter = &rateLimitEntry{
			isBUCP:  true,
			limiter: rate.NewLimiter(l.rate, l.burst),
		}
		l.cache.Set(ip, limiter, cache.DefaultExpiration)
	}
	limiter.(*rateLimitEntry).isBUCP = true
}

type rateLimitEntry struct {
	isBUCP  bool
	limiter *rate.Limiter
}
