package handlers

import (
	"context"
	"log/slog"
	"net/http"

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
