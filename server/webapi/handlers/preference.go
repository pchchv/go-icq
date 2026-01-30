package handlers

import (
	"context"

	"github.com/pchchv/go-icq/state"
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
