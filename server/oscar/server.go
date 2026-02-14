package oscar

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/pchchv/go-icq/config"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
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

type oscarServer struct {
	AuthService
	BuddyListRegistry
	ChatSessionManager
	DepartureNotifier
	Logger *slog.Logger
	OnlineNotifier
	SNACHandler func(ctx context.Context, serverType uint16, instance *state.SessionInstance, inFrame wire.SNACFrame, r io.Reader, rw ResponseWriter, listener config.Listener) error
	RateLimitUpdater
	wire.SNACRateLimits
	*IPRateLimiter
	recalcWarning  func(ctx context.Context, instance *state.SessionInstance) error
	lowerWarnLevel func(ctx context.Context, instance *state.SessionInstance)
}

func (s oscarServer) processFLAPAuth(ctx context.Context, signonFrame wire.FLAPSignonFrame, flapc *wire.FlapClient, advertisedHost string) error {
	tlv, err := s.AuthService.FLAPLogin(ctx, signonFrame, advertisedHost)
	if err != nil {
		return err
	}
	return flapc.NewSignoff(tlv)
}

func (s oscarServer) processBUCPAuth(ctx context.Context, flapc *wire.FlapClient, advertisedHost string) error {
	var frames int
	for {
		frame, err := flapc.ReceiveFLAP()
		if err != nil {
			return err
		}

		if frames > 10 {
			// a lot of frames received, the client is misbehaving
			return fmt.Errorf("too many auth flap packets received")
		}
		frames++

		switch frame.FrameType {
		case wire.FLAPFrameSignoff:
			s.Logger.Debug("signed off mid-login")
			return io.EOF // client disconnected
		case wire.FLAPFrameKeepAlive:
			s.Logger.Debug("received flap keepalive frame")
		case wire.FLAPFrameData:
			buf := bytes.NewReader(frame.Payload)
			fr := wire.SNACFrame{}
			if err := wire.UnmarshalBE(&fr, buf); err != nil {
				return err
			}

			switch {
			case fr.FoodGroup == wire.BUCP && fr.SubGroup == wire.BUCPChallengeRequest:
				challengeRequest := wire.SNAC_0x17_0x06_BUCPChallengeRequest{}
				if err := wire.UnmarshalBE(&challengeRequest, buf); err != nil {
					return err
				}

				outSNAC, err := s.BUCPChallenge(ctx, challengeRequest, uuid.New)
				if err != nil {
					return err
				}

				if err := flapc.SendSNAC(outSNAC.Frame, outSNAC.Body); err != nil {
					return err
				}

				if outSNAC.Frame.SubGroup == wire.BUCPLoginResponse {
					screenName, _ := challengeRequest.String(wire.LoginTLVTagsScreenName)
					s.Logger.Debug("failed BUCP challenge: user does not exist", "screen_name", screenName)
					return nil // account does not exist
				}
			case fr.FoodGroup == wire.BUCP && fr.SubGroup == wire.BUCPLoginRequest:
				loginRequest := wire.SNAC_0x17_0x02_BUCPLoginRequest{}
				if err := wire.UnmarshalBE(&loginRequest, buf); err != nil {
					return err
				}

				outSNAC, err := s.BUCPLogin(ctx, loginRequest, advertisedHost)
				if err != nil {
					return err
				}

				loginResp := outSNAC.Body.(wire.SNAC_0x17_0x03_BUCPLoginResponse)
				// clients expect login response as SNAC on FLAP channel 2 followed by
				// a FLAP signoff frame to properly close the auth connection
				if err := flapc.SendSNAC(outSNAC.Frame, loginResp); err != nil {
					return err
				}

				return flapc.NewSignoff(loginResp.TLVRestBlock)
			default:
				s.Logger.Debug("unexpected SNAC received during login", "foodgroup", wire.FoodGroupName(fr.FoodGroup), "subgroup", wire.SubGroupName(fr.FoodGroup, fr.SubGroup))
				return io.EOF
			}
		default:
			s.Logger.Debug("unexpected frame type received during login", "type", frame.FrameType)
			return io.EOF
		}
	}
}
