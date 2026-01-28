package handlers

import (
	"context"
	"log/slog"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// BuddyBroadcaster broadcasts buddy presence updates.
type BuddyBroadcaster interface {
	BroadcastBuddyArrived(ctx context.Context, screenName state.IdentScreenName, userInfo wire.TLVUserInfo) error
	BroadcastBuddyDeparted(ctx context.Context, instance *state.SessionInstance) error
}

// BuddyPresenceInfo represents presence information for a buddy.
type BuddyPresenceInfo struct {
	AimID      string `json:"aimId" xml:"aimId"`
	State      string `json:"state" xml:"state"` // "online", "offline", "away", "idle"
	AwayMsg    string `json:"awayMsg,omitempty" xml:"awayMsg,omitempty"`
	StatusMsg  string `json:"statusMsg,omitempty" xml:"statusMsg,omitempty"`
	OnlineTime int64  `json:"onlineTime,omitempty" xml:"onlineTime,omitempty"`
	IdleTime   int    `json:"idleTime,omitempty" xml:"idleTime,omitempty"`
	UserType   string `json:"userType" xml:"userType"` // "aim", "icq", "admin"
}

// BuddyGroupInfo represents a buddy group with its members.
type BuddyGroupInfo struct {
	Name    string              `json:"name" xml:"name"`
	Buddies []BuddyPresenceInfo `json:"buddies" xml:"buddies>buddy"`
}

// ProfileManager manages user profiles (uses types.ProfileManager).
type ProfileManager interface {
	SetProfile(ctx context.Context, screenName state.IdentScreenName, profile state.UserProfile) error
	Profile(ctx context.Context, screenName state.IdentScreenName) (state.UserProfile, error)
}

// PresenceData contains presence information.
type PresenceData struct {
	Groups []BuddyGroupInfo    `json:"groups,omitempty" xml:"groups>group,omitempty"`
	Users  []BuddyPresenceInfo `json:"users,omitempty" xml:"users>user,omitempty"`
}

// PresenceHandler handles Web AIM API presence-related endpoints.
type PresenceHandler struct {
	RelationshipFetcher RelationshipFetcher
	SessionRetriever    SessionRetriever
	FeedbagRetriever    FeedbagRetriever
	BuddyBroadcaster    BuddyBroadcaster
	SessionManager      *state.WebAPISessionManager
	ProfileManager      ProfileManager
	Logger              *slog.Logger
}
