package handlers

import (
	"log/slog"

	"github.com/pchchv/go-icq/state"
)

// VanityHandler handles Web AIM API vanity URL endpoints.
type VanityHandler struct {
	SessionManager *state.WebAPISessionManager
	VanityManager  *state.VanityURLManager
	Logger         *slog.Logger
}
