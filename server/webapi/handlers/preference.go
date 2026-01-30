package handlers

import (
	"context"
	"log/slog"
	"net/http"

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
