package handlers

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/pchchv/go-icq/state"
)

// VanityHandler handles Web AIM API vanity URL endpoints.
type VanityHandler struct {
	SessionManager *state.WebAPISessionManager
	VanityManager  *state.VanityURLManager
	Logger         *slog.Logger
}

// GetVanityInfo handles GET /aim/getVanityInfo requests to retrieve vanity URL information.
func (h *VanityHandler) GetVanityInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// according to spec,
	// this endpoint requires signed request parameters but
	// we'll make them optional for compatibility
	ts := r.URL.Query().Get("ts")
	sig := r.URL.Query().Get("sig_sha256")
	// validate timestamp if provided
	if ts != "" && sig == "" {
		SendError(w, http.StatusBadRequest, "signature required when timestamp provided")
		return
	}

	// get authentication from either aimsid or token
	aimsid := r.URL.Query().Get("aimsid")
	// token auth not fully implemented
	_ = r.URL.Query().Get("a")
	var screenName string
	if aimsid != "" {
		if session, err := h.SessionManager.GetSession(r.Context(), aimsid); err == nil {
			screenName = session.ScreenName.String()
		}
	}

	// if no explicit target, use authenticated user
	targetUser := r.URL.Query().Get("t")
	if targetUser == "" {
		if screenName != "" {
			targetUser = screenName
		} else {
			SendError(w, http.StatusBadRequest, "missing target user")
			return
		}
	}

	h.Logger.DebugContext(ctx, "retrieving vanity info",
		"targetUser", targetUser,
		"authenticated", screenName,
	)

	// lookup vanity info by screen name
	info, err := h.VanityManager.GetVanityInfoByScreenName(ctx, targetUser)
	// handle error or no vanity URL found
	if err != nil || info == nil {
		if err != nil && !strings.Contains(err.Error(), "not found") {
			h.Logger.ErrorContext(ctx, "failed to get vanity info",
				"error", err,
			)
			SendError(w, http.StatusInternalServerError, "failed to retrieve vanity info")
			return
		}

		// no vanity URL configured - return not found
		response := BaseResponse{}
		response.Response.StatusCode = 200
		response.Response.StatusText = "OK"
		response.Response.Data = map[string]interface{}{
			"found":      false,
			"screenName": targetUser,
		}
		SendResponse(w, r, response, h.Logger)
		return
	}

	// build response
	responseData := map[string]interface{}{
		"found":      true,
		"screenName": info.ScreenName,
		"vanityUrl":  info.VanityURL,
		"profileUrl": info.ProfileURL,
		"isActive":   info.IsActive,
	}
	// add optional fields if present
	if info.DisplayName != "" {
		responseData["displayName"] = info.DisplayName
	}

	if info.Bio != "" {
		responseData["bio"] = info.Bio
	}

	if info.Location != "" {
		responseData["location"] = info.Location
	}

	if info.Website != "" {
		responseData["website"] = info.Website
	}

	// add extra data if present
	if info.Extra != nil {
		for k, v := range info.Extra {
			responseData[k] = v
		}
	}

	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	response.Response.Data = responseData
	SendResponse(w, r, response, h.Logger)
}
