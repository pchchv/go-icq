package handlers

import "log/slog"

// BuddyListManager handles the conversion of OSCAR feedbag data
// to WebAPI buddy list format for web clients.
type BuddyListManager struct {
	feedbagRetriever FeedbagRetriever
	sessionRetriever SessionRetriever
	logger           *slog.Logger
}

// NewBuddyListManager creates a new instance of the buddy list manager.
func NewBuddyListManager(feedbagRetriever FeedbagRetriever, sessionRetriever SessionRetriever, logger *slog.Logger) *BuddyListManager {
	return &BuddyListManager{
		feedbagRetriever: feedbagRetriever,
		sessionRetriever: sessionRetriever,
		logger:           logger,
	}
}
