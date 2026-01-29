package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/pchchv/go-icq/state"
)

// BuddyFeedHandler handles Web AIM API buddy feed endpoints.
type BuddyFeedHandler struct {
	SessionRetriever SessionRetriever
	SessionManager   *state.WebAPISessionManager
	FeedManager      *state.BuddyFeedManager
	Logger           *slog.Logger
}

// PushFeed handles GET /buddyfeed/pushFeed requests to submit feed updates.
func (h *BuddyFeedHandler) PushFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// get authentication token or session
	token := r.URL.Query().Get("a")
	aimsid := r.URL.Query().Get("aimsid")
	if token == "" && aimsid == "" {
		SendError(w, http.StatusBadRequest, "authentication required")
		return
	}

	var screenName string
	if aimsid != "" {
		session, err := h.SessionManager.GetSession(r.Context(), aimsid)
		if err != nil {
			SendError(w, http.StatusUnauthorized, "invalid or expired session")
			return
		}
		screenName = session.ScreenName.String()

		if err := h.SessionManager.TouchSession(r.Context(), aimsid); err != nil {
			h.Logger.WarnContext(ctx, "failed to touch session", "aimsid", aimsid, "error", err)
		}
	} else {
		// extract screen name from token authentication
		screenName = r.URL.Query().Get("s")
		if screenName == "" {
			SendError(w, http.StatusBadRequest, "missing source user")
			return
		}
	}

	// extract feed parameters as per spec
	feedTitle := r.URL.Query().Get("feedTitle")
	feedLink := r.URL.Query().Get("feedLink")
	feedDesc := r.URL.Query().Get("feedDesc")
	itemTitle := r.URL.Query().Get("itemTitle")
	itemLink := r.URL.Query().Get("itemLink")
	itemGuid := r.URL.Query().Get("itemGuid")
	// validate required parameters
	if feedTitle == "" || feedLink == "" || feedDesc == "" ||
		itemTitle == "" || itemLink == "" || itemGuid == "" {
		SendError(w, http.StatusBadRequest, "missing required feed parameters")
		return
	}

	h.Logger.InfoContext(ctx, "pushing feed update",
		"screenName", screenName,
		"itemTitle", itemTitle,
	)

	// get or create feed for user
	feedID, err := h.FeedManager.GetOrCreateFeedForUser(ctx, screenName, "status")
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get/create feed",
			"screenName", screenName,
			"error", err,
		)
		SendError(w, http.StatusInternalServerError, "failed to get/create feed")
		return
	}

	// build feed item
	item := state.BuddyFeedItem{
		Title:       itemTitle,
		Description: r.URL.Query().Get("itemDesc"),
		Link:        itemLink,
		GUID:        itemGuid,
		Author:      screenName,
		PublishedAt: time.Now(),
	}

	// add category if provided
	if category := r.URL.Query().Get("itemCategory"); category != "" {
		item.Categories = []string{category}
	}

	// add the feed item
	if _, err := h.FeedManager.AddFeedItem(ctx, feedID, item); err != nil {
		h.Logger.ErrorContext(ctx, "failed to add feed item",
			"screenName", screenName,
			"feedID", feedID,
			"error", err,
		)
		SendError(w, http.StatusInternalServerError, "failed to add feed item")
		return
	}

	// send success response
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	response.Response.Data = map[string]interface{}{
		"success": true,
	}

	SendResponse(w, r, response, h.Logger)
}
