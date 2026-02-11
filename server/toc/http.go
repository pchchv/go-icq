package toc

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"

	"github.com/pchchv/go-icq/wire"
	"golang.org/x/net/html"
)

var (
	profileTemplate   *template.Template
	directoryTemplate *template.Template
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

func (s OSCARProxy) logAndReturn500(ctx context.Context, w http.ResponseWriter, err error) {
	s.Logger.ErrorContext(ctx, "internal service error", "err", err.Error())
	http.Error(w, "internal server error", http.StatusInternalServerError)
}

func (s OSCARProxy) outputSearchResults(ctx context.Context, w http.ResponseWriter, users ...wire.TLVBlock) {
	type DirSearchResult struct {
		FirstName  string
		MiddleName string
		LastName   string
		MaidenName string
		Country    string
		State      string
		City       string
		NickName   string
		ZIP        string
		Address    string
		ScreenName string
	}

	type PageData struct {
		Results []DirSearchResult
	}

	results := make([]DirSearchResult, 0, len(users))
	for _, result := range users {
		rec := DirSearchResult{}
		rec.ScreenName, _ = result.String(wire.ODirTLVScreenName)
		rec.FirstName, _ = result.String(wire.ODirTLVFirstName)
		rec.MiddleName, _ = result.String(wire.ODirTLVMiddleName)
		rec.LastName, _ = result.String(wire.ODirTLVLastName)
		rec.MaidenName, _ = result.String(wire.ODirTLVMaidenName)
		rec.Country, _ = result.String(wire.ODirTLVCountry)
		rec.State, _ = result.String(wire.ODirTLVState)
		rec.City, _ = result.String(wire.ODirTLVCity)
		rec.NickName, _ = result.String(wire.ODirTLVNickName)
		rec.ZIP, _ = result.String(wire.ODirTLVZIP)
		rec.Address, _ = result.String(wire.ODirTLVAddress)
		results = append(results, rec)
	}

	if err := directoryTemplate.Execute(w, PageData{Results: results}); err != nil {
		s.logAndReturn500(ctx, w, fmt.Errorf("t.Execute: %w", err))
	}
}

// extractProfile extracts the contents of an HTML <BODY>.
// If there's no HTML body, just return the text.
//
// It only returns the following HTML tags:
//
//	<a>
//	<b>
//	<i>
//	<s>
//	<u>
//	<br>
//	<hr>
//	<sub>
//	<sup>
//	<font>
func extractProfile(htmlContent []byte) string {
	var bodyContent bytes.Buffer
	tokenizer := html.NewTokenizer(bytes.NewReader(htmlContent))
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			if err := tokenizer.Err(); err != nil && err != io.EOF {
				return "unable to read profile"
			}
			return bodyContent.String()
		case html.StartTagToken, html.EndTagToken:
			token := tokenizer.Token()
			switch token.Data {
			case "b", "i", "font", "a", "u", "br", "hr", "s", "sub", "sup":
				bodyContent.WriteString(token.String())
			}
		case html.TextToken:
			bodyContent.Write(tokenizer.Text())
		}
	}
}
