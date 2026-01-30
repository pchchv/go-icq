package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pchchv/go-icq/server/webapi/types"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// WebAPISessionManager provides methods to manage WebAPI sessions.
type WebAPISessionManager interface {
	GetSession(ctx context.Context, aimsid string) (*state.WebAPISession, error)
	TouchSession(ctx context.Context, aimsid string) error
}

// FeedbagManager provides methods to manage buddy lists.
type FeedbagManager interface {
	RetrieveFeedbag(ctx context.Context, screenName state.IdentScreenName) ([]wire.FeedbagItem, error)
	InsertItem(ctx context.Context, screenName state.IdentScreenName, item wire.FeedbagItem) error
	UpdateItem(ctx context.Context, screenName state.IdentScreenName, item wire.FeedbagItem) error
	DeleteItem(ctx context.Context, screenName state.IdentScreenName, item wire.FeedbagItem) error
}

// BuddyListHandler handles Web AIM API buddy list management endpoints.
type BuddyListHandler struct {
	SessionManager WebAPISessionManager
	FeedbagManager FeedbagManager
	Logger         *slog.Logger
}

// AddBuddy handles GET /buddylist/addBuddy requests.
func (h *BuddyListHandler) AddBuddy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// get session ID from parameters
	aimsid := r.URL.Query().Get("aimsid")
	if aimsid == "" {
		h.sendError(w, http.StatusBadRequest, "missing aimsid parameter")
		return
	}

	// get session
	session, err := h.SessionManager.GetSession(r.Context(), aimsid)
	if err != nil {
		switch err {
		case state.ErrNoWebAPISession:
			h.sendError(w, http.StatusNotFound, "session not found")
		case state.ErrWebAPISessionExpired:
			h.sendError(w, http.StatusGone, "session expired")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	// touch the session
	h.SessionManager.TouchSession(r.Context(), aimsid)
	// get buddy and group parameters
	buddyName := strings.TrimSpace(r.URL.Query().Get("buddy"))
	groupName := strings.TrimSpace(r.URL.Query().Get("group"))
	if buddyName == "" {
		h.sendError(w, http.StatusBadRequest, "missing buddy parameter")
		return
	}

	if groupName == "" {
		// default group
		groupName = "Buddies"
	}

	// add buddy to feedbag
	resultCode, buddyInfo := h.addBuddyToFeedbag(ctx, session.ScreenName.IdentScreenName(), buddyName, groupName)
	// prepare response
	responseData := map[string]interface{}{
		"resultCode": resultCode,
	}
	if resultCode == "success" {
		responseData["buddyInfo"] = buddyInfo
	}

	resp := BaseResponse{}
	resp.Response.StatusCode = 200
	resp.Response.StatusText = "OK"
	resp.Response.Data = responseData
	SendResponse(w, r, resp, h.Logger)

	if resultCode == "success" && session.EventQueue != nil {
		// push buddy list update event to the session's event queue
		event := types.BuddyListEvent{
			Action: "add",
			Buddy:  buddyInfo,
			Group:  groupName,
		}
		session.EventQueue.Push(types.EventTypeBuddyList, event)
	}

	h.Logger.InfoContext(ctx, "buddy added",
		"aimsid", aimsid,
		"buddy", buddyName,
		"group", groupName,
		"result", resultCode,
	)
}

// AddTempBuddy handles GET /aim/addTempBuddy requests.
// This adds temporary buddies to the session without persisting them to the feedbag.
// The temporary buddies are only visible for the duration of the session.
func (h *BuddyListHandler) AddTempBuddy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// get session ID from parameters
	aimsid := r.URL.Query().Get("aimsid")
	if aimsid == "" {
		h.sendError(w, http.StatusBadRequest, "missing aimsid parameter")
		return
	}

	// get session
	session, err := h.SessionManager.GetSession(r.Context(), aimsid)
	if err != nil {
		switch err {
		case state.ErrNoWebAPISession:
			h.sendError(w, http.StatusNotFound, "session not found")
		case state.ErrWebAPISessionExpired:
			h.sendError(w, http.StatusGone, "session expired")
		default:
			h.sendError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	// touch the session
	h.SessionManager.TouchSession(r.Context(), aimsid)
	// get buddy names from parameters
	// the WebAPI accepts multiple buddy names via &t= parameters
	buddyNames := r.URL.Query()["t"]
	if len(buddyNames) == 0 {
		h.sendError(w, http.StatusBadRequest, "missing buddy names (t parameter)")
		return
	}

	// store temporary buddies in the session
	// NOTE: These are not persisted to the feedbag database.
	if session.TempBuddies == nil {
		session.TempBuddies = make(map[string]bool)
	}

	for _, buddyName := range buddyNames {
		buddyName = strings.TrimSpace(buddyName)
		if buddyName != "" {
			session.TempBuddies[buddyName] = true
		}
	}

	// prepare response
	responseData := map[string]interface{}{
		"resultCode": "success",
		"buddyNames": buddyNames,
	}

	resp := BaseResponse{}
	resp.Response.StatusCode = 200
	resp.Response.StatusText = "OK"
	resp.Response.Data = responseData
	SendResponse(w, r, resp, h.Logger)

	// push temp buddy event to the session's event queue
	if session.EventQueue != nil {
		for _, buddyName := range buddyNames {
			buddyName = strings.TrimSpace(buddyName)
			if buddyName != "" {
				// create minimal buddy info for temp buddy
				buddyInfo := &BuddyPresenceInfo{
					AimID:    buddyName,
					State:    "offline", // Default state
					UserType: "aim",
				}

				event := types.BuddyListEvent{
					Action: "addTemp",
					Buddy:  buddyInfo,
				}
				session.EventQueue.Push(types.EventTypeBuddyList, event)
			}
		}
	}

	h.Logger.InfoContext(ctx, "temporary buddies added",
		"aimsid", aimsid,
		"buddies", buddyNames,
		"count", len(buddyNames),
	)
}

// addBuddyToFeedbag adds a buddy to the user's feedbag.
func (h *BuddyListHandler) addBuddyToFeedbag(ctx context.Context, screenName state.IdentScreenName, buddyName, groupName string) (string, *BuddyPresenceInfo) {
	// retrieve current feedbag
	items, err := h.FeedbagManager.RetrieveFeedbag(ctx, screenName)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to retrieve feedbag", "err", err.Error())
		return "error", nil
	}

	// check if buddy already exists
	for _, item := range items {
		if item.ClassID == wire.FeedbagClassIdBuddy && item.Name == buddyName {
			return "alreadyExists", nil
		}
	}

	// find or create the group
	var groupID uint16
	var groupFound bool
	var maxGroupID uint16
	for _, item := range items {
		if item.ClassID == wire.FeedbagClassIdGroup {
			if item.ItemID > maxGroupID {
				maxGroupID = item.ItemID
			}

			// check group name
			if item.Name == groupName {
				groupID = item.ItemID
				groupFound = true
			}
		}
	}

	// if group doesn't exist, create it
	if !groupFound {
		groupID = maxGroupID + 1
		groupItem := wire.FeedbagItem{
			ItemID:    groupID,
			ClassID:   wire.FeedbagClassIdGroup,
			Name:      groupName,
			GroupID:   0,
			TLVLBlock: wire.TLVLBlock{},
		}

		if err := h.FeedbagManager.InsertItem(ctx, screenName, groupItem); err != nil {
			h.Logger.ErrorContext(ctx, "failed to create group", "err", err.Error())
			return "error", nil
		}
	}

	// find next available item ID for buddy
	var maxBuddyID uint16
	for _, item := range items {
		if item.ClassID == wire.FeedbagClassIdBuddy && item.ItemID > maxBuddyID {
			maxBuddyID = item.ItemID
		}
	}

	// create buddy item
	buddyItem := wire.FeedbagItem{
		ItemID:    maxBuddyID + 1,
		ClassID:   wire.FeedbagClassIdBuddy,
		Name:      buddyName,
		GroupID:   groupID,
		TLVLBlock: wire.TLVLBlock{},
	}

	// insert buddy into feedbag
	if err := h.FeedbagManager.InsertItem(ctx, screenName, buddyItem); err != nil {
		h.Logger.ErrorContext(ctx, "failed to add buddy", "err", err.Error())
		return "error", nil
	}

	// get current presence for the buddy
	buddyInfo := &BuddyPresenceInfo{
		AimID:    buddyName,
		State:    "offline", // Default to offline
		UserType: "aim",
	}

	return "success", buddyInfo
}

// sendError is a convenience method that wraps the common SendError function.
func (h *BuddyListHandler) sendError(w http.ResponseWriter, statusCode int, message string) {
	SendError(w, statusCode, message)
}
