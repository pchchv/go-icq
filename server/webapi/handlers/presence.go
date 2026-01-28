package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/pchchv/go-icq/server/webapi/types"
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

// broadcastPresenceEvent sends presence updates to all WebAPI sessions watching this user.
func (h *PresenceHandler) broadcastPresenceEvent(screenName state.IdentScreenName, stateStr, awayMsg, statusMsg string) {
	// get all sessions that have this user in their buddy list
	// for now, we'll broadcast to all sessions (this should be optimized)
	// using background context as this is an async broadcast operation
	for _, sess := range h.SessionManager.GetAllSessions(context.Background()) {
		if sess.EventQueue != nil && sess.Events != nil {
			// check if session is subscribed to presence events
			for _, event := range sess.Events {
				if event == "presence" || event == "myInfo" {
					eventData := types.PresenceEvent{
						AimID:     screenName.String(),
						State:     stateStr,
						AwayMsg:   awayMsg,
						StatusMsg: statusMsg,
					}
					sess.EventQueue.Push(types.EventTypePresence, eventData)
					break
				}
			}
		}
	}
}

// sendError is a convenience method that wraps the common SendError function.
func (h *PresenceHandler) sendError(w http.ResponseWriter, statusCode int, message string) {
	SendError(w, statusCode, message)
}
