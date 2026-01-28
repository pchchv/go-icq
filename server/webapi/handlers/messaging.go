package handlers

import (
	"context"
	"log/slog"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// RelationshipFetcher defines methods for fetching user relationships.
type RelationshipFetcher interface {
	Relationship(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) (state.Relationship, error)
}

// OfflineMessageManager defines methods for managing offline messages.
type OfflineMessageManager interface {
	SaveMessage(ctx context.Context, msg state.OfflineMessage) (int, error)
}

// MessageRelayer defines methods for relaying messages between users.
type MessageRelayer interface {
	RelayToScreenName(ctx context.Context, recipient state.IdentScreenName, msg wire.SNACMessage)
}

// MessagingHandler handles Web AIM API messaging endpoints.
type MessagingHandler struct {
	OfflineMessageManager OfflineMessageManager
	RelationshipFetcher   RelationshipFetcher
	SessionRetriever      SessionRetriever
	SessionManager        *state.WebAPISessionManager
	MessageRelayer        MessageRelayer
	Logger                *slog.Logger
}
