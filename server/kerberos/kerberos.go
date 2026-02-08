package kuberos

import (
	"log/slog"
	"net/http"
)

// Server hosts an HTTP endpoint capable of
// handling AIM-style Kerberos authentication.
// The messages are structured as SNACs transmitted over HTTP.
type Server struct {
	servers []*http.Server
	logger  *slog.Logger
}
