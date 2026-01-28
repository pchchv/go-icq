package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/pchchv/go-icq/server/webapi/types"
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

// SetTyping handles the /im/setTyping endpoint for typing indicators.
func (h *MessagingHandler) SetTyping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// get session from aimsid
	aimsid := r.URL.Query().Get("aimsid")
	if aimsid == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "missing required parameter: aimsid")
		return
	}

	sess, err := h.SessionManager.GetSession(r.Context(), aimsid)
	if err != nil {
		if err == state.ErrNoWebAPISession || err == state.ErrWebAPISessionExpired {
			h.sendErrorResponse(w, http.StatusUnauthorized, "invalid or expired session")
		} else {
			h.sendErrorResponse(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	// update session activity
	if err := h.SessionManager.TouchSession(r.Context(), aimsid); err != nil {
		h.Logger.WarnContext(ctx, "failed to touch session", "aimsid", aimsid, "error", err)
	}

	// parse parameters
	recipient := r.URL.Query().Get("t")
	if recipient == "" {
		h.sendErrorResponse(w, http.StatusBadRequest, "missing required parameter: t (recipient)")
		return
	}

	typingStr := r.URL.Query().Get("typing")
	typing := false
	if typingStr != "" {
		var err error
		typing, err = strconv.ParseBool(typingStr)
		if err != nil {
			// Try numeric format (0/1)
			typing = typingStr == "1"
		}
	}

	// create recipient identifier
	recipientIdent := state.NewIdentScreenName(recipient)
	// check blocking relationship
	rel, err := h.RelationshipFetcher.Relationship(ctx, sess.ScreenName.IdentScreenName(), recipientIdent)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to fetch relationship", "error", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// check if sender blocks recipient or recipient blocks sender
	if rel.BlocksYou || rel.YouBlock {
		// either party blocks the other - silently succeed without sending notification
		h.sendSuccessResponse(w, r, nil)
		return
	}

	// check if recipient is online
	recipientSession := h.SessionRetriever.RetrieveSession(recipientIdent)
	if recipientSession == nil {
		// silently succeed even if recipient is offline
		h.sendSuccessResponse(w, r, nil)
		return
	}

	// generate typing notification cookie
	var cookie [8]byte
	if _, err := rand.Read(cookie[:]); err != nil {
		h.Logger.ErrorContext(ctx, "failed to generate typing cookie", "error", err)
		h.sendErrorResponse(w, http.StatusInternalServerError, "internal server error")
		return
	}
	cookieUint64 := binary.BigEndian.Uint64(cookie[:])

	// create typing notification
	var notificationType uint16
	if typing {
		notificationType = 0x0002 // typing started
	} else {
		notificationType = 0x0001 // typing stopped
	}

	typingNotification := wire.SNAC_0x04_0x14_ICBMClientEvent{
		Cookie:     cookieUint64,
		ChannelID:  wire.ICBMChannelIM,
		ScreenName: sess.ScreenName.String(),
		Event:      notificationType,
	}

	// send typing notification to recipient
	h.MessageRelayer.RelayToScreenName(ctx, recipientIdent, wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.ICBM,
			SubGroup:  wire.ICBMClientEvent,
			RequestID: wire.ReqIDFromServer,
		},
		Body: typingNotification,
	})

	// queue typing event for the recipient's WebAPI session if they have one
	if recipientWebSession, err := h.SessionManager.GetSessionByUser(r.Context(), recipientIdent); err == nil && recipientWebSession != nil {
		eventData := types.TypingEvent{
			From:   sess.ScreenName.String(),
			Typing: typing,
		}
		recipientWebSession.EventQueue.Push(types.EventTypeTyping, eventData)
	}

	h.Logger.DebugContext(ctx, "sent typing notification", "from", sess.ScreenName.String(), "to", recipient, "typing", typing)
	// send success response
	h.sendSuccessResponse(w, r, nil)
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

// encodeIMMessage encodes a text message into the OSCAR IM format.
func (h *MessagingHandler) encodeIMMessage(text string, autoResponse bool) []byte {
	// create ICBM fragment list for the message
	frags, err := wire.ICBMFragmentList(text)
	if err != nil {
		// fragment creation fails
		// return simple text bytes
		return []byte(text)
	}

	// marshal the fragments
	buf := &bytes.Buffer{}
	for _, frag := range frags {
		if err := wire.MarshalBE(frag, buf); err != nil {
			// marshaling fails
			// return simple text bytes
			return []byte(text)
		}
	}

	return buf.Bytes()
}
