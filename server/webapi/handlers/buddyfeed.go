package handlers

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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

// GetUser handles GET /buddyfeed/getUser requests to retrieve a user's feed.
func (h *BuddyFeedHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// get target user from 't' parameter as per spec
	targetUser := r.URL.Query().Get("t")
	if targetUser == "" {
		SendError(w, http.StatusBadRequest, "missing 't' parameter")
		return
	}

	// get format parameter
	format := strings.ToLower(r.URL.Query().Get("f"))
	h.Logger.DebugContext(ctx, "retrieving user feed",
		"user", targetUser,
		"format", format,
	)

	// get the feed configuration
	feed, err := h.FeedManager.GetUserFeed(ctx, targetUser)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get user feed",
			"user", targetUser,
			"error", err,
		)
		SendError(w, http.StatusInternalServerError, "failed to retrieve feed")
		return
	}

	// build feed response
	var feedResponse *FeedResponse
	if feed == nil {
		// no feed configured
		// generate empty feed
		feedResponse = GenerateEmptyFeed(targetUser)
	} else {
		// get feed items
		items, err := h.FeedManager.GetFeedItems(ctx, feed.ID, 50)
		if err != nil {
			h.Logger.ErrorContext(ctx, "failed to get feed items",
				"feedID", feed.ID,
				"error", err,
			)
			SendError(w, http.StatusInternalServerError, "failed to retrieve feed items")
			return
		}

		feedResponse = &FeedResponse{
			Feed:  *feed,
			Items: items,
		}
	}

	// send response based on format
	switch format {
	case "atom":
		h.sendAtomFeed(w, feedResponse.ToAtom())
	case "json":
		response := BaseResponse{}
		response.Response.StatusCode = 200
		response.Response.StatusText = "OK"
		response.Response.Data = feedResponse.ToJSON()
		SendResponse(w, r, response, h.Logger)
	case "rss", "native", "":
		h.sendRSSFeed(w, feedResponse.ToRSS())
	default:
		h.sendRSSFeed(w, feedResponse.ToRSS())
	}
}

// GetBuddylist handles GET /buddyfeed/getBuddylist requests to retrieve aggregated buddy feeds.
func (h *BuddyFeedHandler) GetBuddylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// authentication required for buddy list feed
	aimsid := r.URL.Query().Get("aimsid")
	if aimsid == "" {
		SendError(w, http.StatusBadRequest, "missing aimsid parameter")
		return
	}

	// get session
	session, err := h.SessionManager.GetSession(r.Context(), aimsid)
	if err != nil {
		SendError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}

	// update session activity
	if err := h.SessionManager.TouchSession(r.Context(), aimsid); err != nil {
		h.Logger.WarnContext(ctx, "failed to touch session", "aimsid", aimsid, "error", err)
	}

	// get format and limit parameters
	format := strings.ToLower(r.URL.Query().Get("f"))
	if format == "" {
		format = "rss"
	}

	// default limit
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 500 {
				// max limit
				limit = 500
			}
		}
	}

	h.Logger.DebugContext(ctx, "retrieving buddy list feed",
		"screenName", session.ScreenName.String(),
		"format", format,
		"limit", limit,
	)

	// get buddy list from OSCAR session if available
	var buddies []state.IdentScreenName
	// return empty feed for now - buddy list integration pending
	if len(buddies) == 0 {
		h.Logger.InfoContext(ctx, "no buddies found for feed aggregation",
			"screenName", session.ScreenName.String(),
		)

		// return empty feed
		emptyFeed := map[string]interface{}{
			"title":       fmt.Sprintf("%s's Buddy Feed", session.ScreenName.String()),
			"description": "Aggregated feed from your buddy list",
			"items":       []interface{}{},
		}

		if format == "json" {
			response := BaseResponse{}
			response.Response.StatusCode = 200
			response.Response.StatusText = "OK"
			response.Response.Data = emptyFeed
			SendResponse(w, r, response, h.Logger)
		} else {
			h.sendEmptyRSSFeed(w, session.ScreenName.String())
		}
		return
	}

	// get aggregated feed items
	items, err := h.FeedManager.GetBuddyListFeedItems(ctx, buddies, limit)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get buddy list feed",
			"screenName", session.ScreenName.String(),
			"error", err,
		)
		SendError(w, http.StatusInternalServerError, "failed to retrieve feed")
		return
	}

	// build aggregated feed response
	feedResponse := &FeedResponse{
		Feed: state.BuddyFeed{
			Title:       "Buddy List Feed",
			Description: "Aggregated feed from your buddy list",
			Link:        "/buddyfeed/getBuddylist",
			PublishedAt: time.Now(),
			UpdatedAt:   time.Now(),
		},
		Items: items,
	}

	// send response based on format
	switch format {
	case "atom":
		h.sendAtomFeed(w, feedResponse.ToAtom())
	case "json":
		response := BaseResponse{}
		response.Response.StatusCode = 200
		response.Response.StatusText = "OK"
		response.Response.Data = feedResponse.ToJSON()
		SendResponse(w, r, response, h.Logger)
	case "rss", "":
		h.sendRSSFeed(w, feedResponse.ToRSS())
	default:
		h.sendRSSFeed(w, feedResponse.ToRSS())
	}
}

// sendRSSFeed sends an RSS feed response.
func (h *BuddyFeedHandler) sendRSSFeed(w http.ResponseWriter, feed *RSSFeed) {
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	// add XML declaration
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
	// marshal and write the feed
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(feed); err != nil {
		h.Logger.Error("failed to encode RSS feed", "error", err)
	}
}

// sendEmptyRSSFeed sends an empty RSS feed.
func (h *BuddyFeedHandler) sendEmptyRSSFeed(w http.ResponseWriter, screenName string) {
	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	emptyFeed := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>%s's Buddy Feed</title>
    <description>Aggregated feed from your buddy list</description>
    <link>/buddyfeed/getBuddylist</link>
    <language>en-US</language>
  </channel>
</rss>`, screenName)
	w.Write([]byte(emptyFeed))
}

// sendAtomFeed sends an Atom feed response.
func (h *BuddyFeedHandler) sendAtomFeed(w http.ResponseWriter, feed *AtomFeed) {
	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	// add XML declaration
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>`))
	// marshal and write the feed
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(feed); err != nil {
		h.Logger.Error("failed to encode Atom feed", "error", err)
	}
}
