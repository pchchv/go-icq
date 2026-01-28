package handlers

import (
	"context"
	"log/slog"
	"net/http"

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

// sendSuccessResponse sends a success response in Web AIM API format.
func (h *MessagingHandler) sendSuccessResponse(w http.ResponseWriter, r *http.Request, data interface{}) {
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	response.Response.Data = data
	SendResponse(w, r, response, h.Logger)
}

// sendErrorResponse sends an error response in Web AIM API format.
func (h *MessagingHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, errorText string) {
	SendError(w, statusCode, errorText)
}
