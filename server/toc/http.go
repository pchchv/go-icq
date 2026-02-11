package toc

import (
	"encoding/hex"
	"net"
	"net/http"
)

// AuthMiddleware is an HTTP middleware that enforces authentication using an
// authorization cookie provided as a query parameter.
// It validates and decrypts the cookie before allowing the request to proceed.
//
// If the `cookie` query parameter is missing or invalid,
// the middleware responds with an appropriate HTTP error:
//   - 400 Bad Request if the `cookie` parameter is missing.
//   - 403 Forbidden if the cookie is invalid or cannot be decrypted.
//
// Requests with a valid cookie are passed to the next handler.
func (s OSCARProxy) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cookie := r.URL.Query().Get("cookie")
		if cookie == "" {
			http.Error(w, "required `cookie` param is missing", http.StatusBadRequest)
			return
		}

		data, err := hex.DecodeString(cookie)
		if err != nil {
			s.Logger.DebugContext(ctx, "error decoding string", "err", err.Error())
			http.Error(w, "invalid auth cookie", http.StatusForbidden)
			return
		}

		if _, err = s.CookieBaker.Crack(data); err != nil {
			s.Logger.DebugContext(ctx, "error cracking auth cookie", "err", err.Error())
			http.Error(w, "invalid auth cookie", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s OSCARProxy) RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			s.Logger.Error("failed to parse remote address", "err", err.Error())
			return
		}

		if !s.HTTPIPRateLimiter.Allow(ip) {
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
