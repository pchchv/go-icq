package handlers

import (
	"log/slog"

	"github.com/pchchv/go-icq/state"
)

// ChatHandler handles Web API chat endpoints.
type ChatHandler struct {
	SessionManager *state.WebAPISessionManager
	ChatManager    *state.WebAPIChatManager
	Logger         *slog.Logger
}
