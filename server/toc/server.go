package toc

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pchchv/go-icq/state"
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

// Server implements a TOC protocol server that multiplexes TOC/HTTP and TOC/FLAP requests.
// It acts as a gateway, forwarding all TOC requests to the OSCAR server for processing.
type Server struct {
	bosProxy           OSCARProxy
	logger             *slog.Logger
	loginIPRateLimiter *IPRateLimiter
	lowerWarnLevel     func(ctx context.Context, instance *state.SessionInstance)
	recalcWarning      func(ctx context.Context, instance *state.SessionInstance) error
	listenerCfg        []string
	listeners          []net.Listener
	servers            []*http.Server
	connMu             sync.Mutex
	conns              map[net.Conn]struct{}
	connWg             sync.WaitGroup
	listenWg           sync.WaitGroup
	shutdownCtx        context.Context
	shutdownCancel     context.CancelFunc
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Debug("Initiating graceful shutdown...")
	s.shutdownCancel()
	s.cleanupListeners()
	// wait for handlers to complete
	done := make(chan struct{})
	go func() {
		s.connWg.Wait()
		s.listenWg.Wait()
		close(done)
	}()

	for _, srv := range s.servers {
		_ = srv.Shutdown(ctx)
	}

	select {
	case <-done:
		s.logger.Info("shutdown complete")
	case <-ctx.Done():
		s.logger.Info("shutdown complete, but connections didn't close cleanly")
	}

	return nil
}

func (s *Server) cleanupListeners() {
	for _, ln := range s.listeners {
		_ = ln.Close()
	}
	s.listeners = nil
}

// channelListener is an implementation of net.Listener that
// accepts connections from a channel instead of a network socket.
// It is useful for attaching an HTTP service to a connection on the fly.
type channelListener struct {
	ch  chan net.Conn // channel used to receive connections
	ctx context.Context
}

// Addr returns the network address of the listener.
// Since channelListener is not bound to a real network address, it returns nil.
func (l *channelListener) Addr() net.Addr {
	return nil
}

// Accept waits for and returns the next connection from the channel.
// If the channel is closed, it returns io.EOF to indicate no more connections.
func (l *channelListener) Accept() (net.Conn, error) {
	select {
	case <-l.ctx.Done():
		return nil, io.EOF
	case ch := <-l.ch:
		return ch, nil
	}
}

// Close closes the listener.
// Since channelListener does not manage an actual network connection,
// this is a no-op and always returns nil.
func (l *channelListener) Close() error {
	return nil
}
