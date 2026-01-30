package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// PermitDenyData contains permit/deny list information.
type PermitDenyData struct {
	PDMode     int      `json:"pdMode" xml:"pdMode"`
	DenyList   []string `json:"denyList,omitempty" xml:"denyList>user,omitempty"`
	PermitList []string `json:"permitList,omitempty" xml:"permitList>user,omitempty"`
}

// PreferenceManager provides methods to manage user preferences.
type PreferenceManager interface {
	SetPreferences(ctx context.Context, screenName state.IdentScreenName, prefs map[string]interface{}) error
	GetPreferences(ctx context.Context, screenName state.IdentScreenName) (map[string]interface{}, error)
}

// PermitDenyManager provides methods to manage permit/deny lists.
type PermitDenyManager interface {
	SetPDMode(ctx context.Context, screenName state.IdentScreenName, mode wire.FeedbagPDMode) error
	GetPDMode(ctx context.Context, screenName state.IdentScreenName) (wire.FeedbagPDMode, error)
	GetPermitList(ctx context.Context, screenName state.IdentScreenName) ([]state.IdentScreenName, error)
	GetDenyList(ctx context.Context, screenName state.IdentScreenName) ([]state.IdentScreenName, error)
	AddPermitBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	RemovePermitBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	AddDenyBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	RemoveDenyBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
}

// PreferenceHandler handles Web AIM API preference-related endpoints.
type PreferenceHandler struct {
	PreferenceManager PreferenceManager
	PermitDenyManager PermitDenyManager
	SessionManager    *state.WebAPISessionManager
	Logger            *slog.Logger
}

// GetPermitDeny handles GET /preference/getPermitDeny requests to retrieve permit/deny settings.
func (h *PreferenceHandler) GetPermitDeny(w http.ResponseWriter, r *http.Request) {
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
		h.sendError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}

	// update session activity
	if err := h.SessionManager.TouchSession(r.Context(), aimsid); err != nil {
		h.Logger.WarnContext(ctx, "failed to touch session", "aimsid", aimsid, "error", err)
	}

	// get PD data
	pdMode, _ := h.PermitDenyManager.GetPDMode(ctx, session.ScreenName.IdentScreenName())
	permitList, _ := h.PermitDenyManager.GetPermitList(ctx, session.ScreenName.IdentScreenName())
	denyList, _ := h.PermitDenyManager.GetDenyList(ctx, session.ScreenName.IdentScreenName())
	// convert to string arrays
	permitUsers := make([]string, len(permitList))
	for i, u := range permitList {
		permitUsers[i] = u.String()
	}

	denyUsers := make([]string, len(denyList))
	for i, u := range denyList {
		denyUsers[i] = u.String()
	}

	h.Logger.DebugContext(ctx, "permit/deny settings retrieved",
		"screenName", session.ScreenName.String(),
		"pdMode", pdMode,
		"permitCount", len(permitUsers),
		"denyCount", len(denyUsers),
	)
	// send response
	permitDenyData := PermitDenyData{
		PDMode:     int(pdMode),
		PermitList: permitUsers,
		DenyList:   denyUsers,
	}
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	response.Response.Data = permitDenyData
	SendResponse(w, r, response, h.Logger)
}

// GetPreferences handles GET /preference/get requests to retrieve user preferences.
func (h *PreferenceHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
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
		h.sendError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}

	// update session activity
	if err := h.SessionManager.TouchSession(r.Context(), aimsid); err != nil {
		h.Logger.WarnContext(ctx, "failed to touch session", "aimsid", aimsid, "error", err)
	}

	// get target user (optional, defaults to session user)
	targetUser := session.ScreenName.IdentScreenName()
	if t := r.URL.Query().Get("t"); t != "" {
		targetUser = state.NewIdentScreenName(t)
	}

	// get all stored preferences or defaults
	allPrefs, err := h.PreferenceManager.GetPreferences(ctx, targetUser)
	if err != nil {
		h.Logger.ErrorContext(ctx, "failed to get preferences", "err", err.Error())
		allPrefs = h.getDefaultPreferences()
	}

	if len(allPrefs) == 0 {
		allPrefs = h.getDefaultPreferences()
	}

	// check if specific preferences are being requested
	requestedPrefs := make(map[string]interface{})
	defaultPrefs := h.getDefaultPreferences()
	// check each known preference key in the query parameters
	// when a preference appears in the query (e.g., playIMSound=1),
	// the client is requesting that specific preference value
	for key := range defaultPrefs {
		if r.URL.Query().Has(key) {
			// client is requesting this specific preference
			if prefValue, exists := allPrefs[key]; exists {
				requestedPrefs[key] = prefValue
			} else {
				requestedPrefs[key] = defaultPrefs[key]
			}
		}
	}

	// if no specific preferences were requested, return all
	var prefs map[string]interface{}
	if len(requestedPrefs) > 0 {
		prefs = requestedPrefs
	} else {
		prefs = allPrefs
	}

	h.Logger.DebugContext(ctx, "preferences retrieved",
		"screenName", targetUser.String(),
		"prefCount", len(prefs),
		"requested", len(requestedPrefs) > 0,
	)
	// check for AMF format to handle special Gromit compatibility requirements
	format := strings.ToLower(r.URL.Query().Get("f"))
	if format == "amf" || format == "amf3" {
		// convert string "1"/"0" to numeric values for Gromit compatibility
		// Gromit expects numeric values for boolean preferences
		convertedPrefs := make(map[string]interface{})
		for key, val := range prefs {
			switch v := val.(type) {
			case string:
				switch v {
				case "1":
					convertedPrefs[key] = 1
				case "0":
					convertedPrefs[key] = 0
				}
			default:
				// keep non-boolean values as strings
				convertedPrefs[key] = val
			}
		}

		prefs = convertedPrefs
		h.Logger.DebugContext(ctx, "AMF preference response",
			"prefs", prefs,
			"prefCount", len(prefs),
			"format", format,
		)

		// ensure prefs is never nil or empty for Gromit
		if len(prefs) == 0 {
			// if no preferences found, at least return the requested ones with defaults
			if len(requestedPrefs) > 0 {
				prefs = requestedPrefs
			} else {
				// return playIMSound as default if nothing else
				prefs = map[string]interface{}{
					"playIMSound": 1,
				}
			}
		}

		// for single preference requests, return directly for Gromit compatibility
		// for multiple preferences, wrap in jsonData
		if len(prefs) != 1 {
			prefs = map[string]interface{}{
				"jsonData": prefs,
			}
		}
	}

	// send response in requested format
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	response.Response.Data = prefs
	SendResponse(w, r, response, h.Logger)
}

// SetPermitDeny handles GET /preference/setPermitDeny requests to update permit/deny settings.
func (h *PreferenceHandler) SetPermitDeny(w http.ResponseWriter, r *http.Request) {
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
		h.sendError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}

	// update session activity
	if err := h.SessionManager.TouchSession(r.Context(), aimsid); err != nil {
		h.Logger.WarnContext(ctx, "failed to touch session", "aimsid", aimsid, "error", err)
	}

	// get pdMode parameter
	if pdModeStr := r.URL.Query().Get("pdMode"); pdModeStr != "" {
		pdMode, err := strconv.Atoi(pdModeStr)
		if err != nil || pdMode < 0 || pdMode > 5 {
			h.sendError(w, http.StatusBadRequest, "invalid pdMode value (must be 0-5)")
			return
		}

		// set the PD mode
		if err = h.PermitDenyManager.SetPDMode(ctx, session.ScreenName.IdentScreenName(), wire.FeedbagPDMode(pdMode)); err != nil {
			h.Logger.ErrorContext(ctx, "failed to set PD mode", "err", err.Error())
			h.sendError(w, http.StatusInternalServerError, "failed to update PD mode")
			return
		}
	}

	// handle permit list updates
	if permitAdd := r.URL.Query().Get("permitAdd"); permitAdd != "" {
		users := strings.Split(permitAdd, ",")
		for _, user := range users {
			if user = strings.TrimSpace(user); user != "" {
				targetSN := state.NewIdentScreenName(user)
				if err := h.PermitDenyManager.AddPermitBuddy(ctx, session.ScreenName.IdentScreenName(), targetSN); err != nil {
					h.Logger.ErrorContext(ctx, "failed to add to permit list", "user", user, "err", err.Error())
				}
			}
		}
	}

	if permitRemove := r.URL.Query().Get("permitRemove"); permitRemove != "" {
		users := strings.Split(permitRemove, ",")
		for _, user := range users {
			if user = strings.TrimSpace(user); user != "" {
				targetSN := state.NewIdentScreenName(user)
				if err := h.PermitDenyManager.RemovePermitBuddy(ctx, session.ScreenName.IdentScreenName(), targetSN); err != nil {
					h.Logger.ErrorContext(ctx, "failed to remove from permit list", "user", user, "err", err.Error())
				}
			}
		}
	}

	// handle deny list updates
	if denyAdd := r.URL.Query().Get("denyAdd"); denyAdd != "" {
		users := strings.Split(denyAdd, ",")
		for _, user := range users {
			if user = strings.TrimSpace(user); user != "" {
				targetSN := state.NewIdentScreenName(user)
				if err := h.PermitDenyManager.AddDenyBuddy(ctx, session.ScreenName.IdentScreenName(), targetSN); err != nil {
					h.Logger.ErrorContext(ctx, "failed to add to deny list", "user", user, "err", err.Error())
				}
			}
		}
	}

	if denyRemove := r.URL.Query().Get("denyRemove"); denyRemove != "" {
		users := strings.Split(denyRemove, ",")
		for _, user := range users {
			if user = strings.TrimSpace(user); user != "" {
				targetSN := state.NewIdentScreenName(user)
				if err := h.PermitDenyManager.RemoveDenyBuddy(ctx, session.ScreenName.IdentScreenName(), targetSN); err != nil {
					h.Logger.ErrorContext(ctx, "failed to remove from deny list", "user", user, "err", err.Error())
				}
			}
		}
	}

	// get updated PD data
	pdMode, _ := h.PermitDenyManager.GetPDMode(ctx, session.ScreenName.IdentScreenName())
	permitList, _ := h.PermitDenyManager.GetPermitList(ctx, session.ScreenName.IdentScreenName())
	denyList, _ := h.PermitDenyManager.GetDenyList(ctx, session.ScreenName.IdentScreenName())
	// convert to string arrays
	permitUsers := make([]string, len(permitList))
	for i, u := range permitList {
		permitUsers[i] = u.String()
	}

	denyUsers := make([]string, len(denyList))
	for i, u := range denyList {
		denyUsers[i] = u.String()
	}

	h.Logger.DebugContext(ctx, "permit/deny settings updated",
		"screenName", session.ScreenName.String(),
		"pdMode", pdMode,
		"permitCount", len(permitUsers),
		"denyCount", len(denyUsers),
	)
	// NOTE: We don't broadcast immediate presence changes here.
	// The blocking relationship is now in the database and will be
	// respected by all future presence checks and message routing.
	// The blocked users will appear offline to each other on the next presence update.

	// send response
	permitDenyData := PermitDenyData{
		PDMode:     int(pdMode),
		PermitList: permitUsers,
		DenyList:   denyUsers,
	}
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	response.Response.Data = permitDenyData
	SendResponse(w, r, response, h.Logger)
}

// SetPreferences handles GET /preference/set requests to update user preferences.
func (h *PreferenceHandler) SetPreferences(w http.ResponseWriter, r *http.Request) {
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
		h.sendError(w, http.StatusUnauthorized, "invalid or expired session")
		return
	}

	// update session activity
	if err := h.SessionManager.TouchSession(r.Context(), aimsid); err != nil {
		h.Logger.WarnContext(ctx, "failed to touch session", "aimsid", aimsid, "error", err)
	}

	// parse preferences from query parameters
	prefs := make(map[string]interface{})
	// common preference keys from the Web AIM API spec
	prefKeys := []string{
		"statusMsg", "awayMsg", "profileMsg", "buddyIcon",
		"soundsOn", "alertsOn", "typingStatus", "idleTime",
		"pdMode", "invisibleTo", "visibleTo", "blockList",
		"allowList", "language", "timeZone", "dateFormat",
		"showTimestamps", "fontSize", "fontFamily", "theme",
		"autoResponse", "saveHistory", "encryptMessages",
	}

	// extract preferences from query parameters
	for _, key := range prefKeys {
		if val := r.URL.Query().Get(key); val != "" {
			// try to parse as boolean
			if val == "true" || val == "false" {
				prefs[key] = val == "true"
			} else if num, err := strconv.Atoi(val); err == nil {
				// try to parse as integer
				prefs[key] = num
			} else {
				// store as string
				prefs[key] = val
			}
		}
	}

	// allow any other parameters starting with "pref_" for extensibility
	for key, values := range r.URL.Query() {
		if strings.HasPrefix(key, "pref_") && len(values) > 0 {
			actualKey := strings.TrimPrefix(key, "pref_")
			prefs[actualKey] = values[0]
		}
	}

	// save preferences
	if err := h.PreferenceManager.SetPreferences(ctx, session.ScreenName.IdentScreenName(), prefs); err != nil {
		h.Logger.ErrorContext(ctx, "failed to set preferences", "err", err.Error())
		h.sendError(w, http.StatusInternalServerError, "failed to save preferences")
		return
	}

	h.Logger.DebugContext(ctx, "preferences updated", "screenName", session.ScreenName.String(), "prefCount", len(prefs))
	// send success response
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	response.Response.Data = prefs
	SendResponse(w, r, response, h.Logger)
}

// getDefaultPreferences returns default preference values that clients expect.
func (h *PreferenceHandler) getDefaultPreferences() map[string]interface{} {
	return map[string]interface{}{
		"autoPlay":            "1",
		"playIMSound":         "1",
		"playBuddySound":      "1",
		"showTimestamps":      "1",
		"showAdsFlag":         "1",
		"soundSetting":        "1",
		"awayMessageOn":       "0",
		"awayMessage":         "",
		"confirmSignOff":      "0",
		"skipNavigator":       "1",
		"displayIdleTime":     "1",
		"repliesAnyone":       "0",
		"repliesUsersOnline":  "0",
		"repliesBuddies":      "0",
		"replyMessage":        "",
		"allowAccessPresence": "0",
		"blockIdleStatus":     "0",
		"reportIdleTyping":    "1",
		"smileysDisabled":     "0",
		"sortBuddiesAlpha":    "0",
		"statusMsg":           "",
		"statusIcon":          "",
		"skin":                "default",
	}
}

// sendError sends an error response in Web AIM API format.
func (h *PreferenceHandler) sendError(w http.ResponseWriter, statusCode int, message string) {
	SendError(w, statusCode, message)
}
