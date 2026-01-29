package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/pchchv/go-icq/state"
)

// ChatHandler handles Web API chat endpoints.
type ChatHandler struct {
	SessionManager *state.WebAPISessionManager
	ChatManager    *state.WebAPIChatManager
	Logger         *slog.Logger
}

// CreateAndJoinChat creates (if needed) and joins a chat room
// GET /chat/createAndJoinChat
func (h *ChatHandler) CreateAndJoinChat(w http.ResponseWriter, r *http.Request) {
	// extract parameters
	aimsid := r.URL.Query().Get("aimsid")
	roomID := r.URL.Query().Get("roomId")
	roomName := r.URL.Query().Get("roomName")
	// validate session
	session, err := h.SessionManager.GetSession(r.Context(), aimsid)
	if err != nil {
		h.Logger.Error("invalid session", "aimsid", aimsid, "error", err)
		SendError(w, http.StatusUnauthorized, "Authentication Required")
		return
	}

	// validate parameters
	// exactly one of roomId or roomName must be provided
	if (roomID == "" && roomName == "") || (roomID != "" && roomName != "") {
		SendError(w, http.StatusBadRequest, "Exactly one of roomId or roomName must be provided")
		return
	}

	// create or join the chat room
	chatSession, room, err := h.ChatManager.CreateAndJoinChat(r.Context(), aimsid, roomID, roomName, string(session.ScreenName))
	if err != nil {
		h.Logger.Error("failed to create/join chat", "error", err, "aimsid", aimsid)
		// determine appropriate error code
		statusCode := http.StatusInternalServerError
		message := "Internal Server Error"
		if strings.Contains(err.Error(), "maximum capacity") {
			statusCode = http.StatusServiceUnavailable
			message = "Room is at maximum capacity"
		} else if strings.Contains(err.Error(), "must be provided") {
			statusCode = http.StatusBadRequest
			message = err.Error()
		}

		SendError(w, statusCode, message)
		return
	}

	// build response
	roomData := map[string]interface{}{
		"roomName":    room.RoomName,
		"roomId":      room.RoomID,
		"instanceId":  room.InstanceID,
		"description": room.Description,
		"roomType":    string(room.RoomType),
	}

	// add category ID if present
	if room.CategoryID != "" {
		roomData["categoryId"] = room.CategoryID
	}

	response := BaseResponse{
		Response: ResponseBody{
			StatusCode: 200,
			StatusText: "OK",
			Data: map[string]interface{}{
				"chatsid": chatSession.ChatSID,
				"room":    roomData,
			},
		},
	}

	// send response
	SendResponse(w, r, response, h.Logger)
	h.Logger.Info("user joined chat room", "screenName", session.ScreenName, "roomName", room.RoomName, "roomID", room.RoomID, "chatsid", chatSession.ChatSID)
}

// LeaveChat leaves the current chat room
// GET /chat/leaveChat
func (h *ChatHandler) LeaveChat(w http.ResponseWriter, r *http.Request) {
	// extract parameters
	aimsid := r.URL.Query().Get("aimsid")
	chatsid := r.URL.Query().Get("chatsid")
	// validate session
	session, err := h.SessionManager.GetSession(r.Context(), aimsid)
	if err != nil {
		h.Logger.Error("invalid session", "aimsid", aimsid, "error", err)
		SendError(w, http.StatusUnauthorized, "Authentication Required")
		return
	}

	// validate required parameters
	if chatsid == "" {
		SendError(w, http.StatusBadRequest, "chatsid is required")
		return
	}

	// leave the chat room
	if err = h.ChatManager.LeaveChat(r.Context(), chatsid); err != nil {
		h.Logger.Error("failed to leave chat", "error", err, "chatsid", chatsid)
		// determine appropriate error code
		statusCode := http.StatusInternalServerError
		message := "Internal Server Error"
		if strings.Contains(err.Error(), "invalid chat session") {
			statusCode = http.StatusNotFound
			message = "Chat session not found"
		}

		SendError(w, statusCode, message)
		return
	}

	// build response
	response := BaseResponse{
		Response: ResponseBody{
			StatusCode: 200,
			StatusText: "OK",
			Data:       map[string]interface{}{},
		},
	}

	// send response
	SendResponse(w, r, response, h.Logger)
	h.Logger.Info("user left chat room", "screenName", session.ScreenName, "chatsid", chatsid)
}
