package toc

import (
	"context"
	"net"
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

// Allow returns true if the request from the
// given IP is allowed under its rate limit.
// If no limiter exists for the IP,
// one is created and tracked in the cache.
func (l *IPRateLimiter) Allow(ip string) (allowed bool) {
	limiter, found := l.cache.Get(ip)
	if !found {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.cache.Set(ip, limiter, cache.DefaultExpiration)
	}

	return limiter.(*rate.Limiter).Allow()
}

// channelListener is an implementation of net.Listener that
// accepts connections from a channel instead of a network socket.
// It is useful for attaching an HTTP service to a connection on the fly.
type channelListener struct {
	ch  chan net.Conn // channel used to receive connections
	ctx context.Context
}
