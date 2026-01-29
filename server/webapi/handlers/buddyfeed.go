package handlers

import (
	"log/slog"

	"github.com/pchchv/go-icq/state"
)

// BuddyFeedHandler handles Web AIM API buddy feed endpoints.
type BuddyFeedHandler struct {
	SessionRetriever SessionRetriever
	SessionManager   *state.WebAPISessionManager
	FeedManager      *state.BuddyFeedManager
	Logger           *slog.Logger
}
