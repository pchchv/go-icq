package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

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

// Icon handles GET /presence/icon requests for presence icons.
func (h *PresenceHandler) Icon(w http.ResponseWriter, r *http.Request) {
	// get parameters
	name := r.URL.Query().Get("name")
	size := r.URL.Query().Get("size")
	iconType := r.URL.Query().Get("type")
	if name == "" {
		h.sendError(w, http.StatusBadRequest, "missing name parameter")
		return
	}

	// default values
	if size == "" {
		size = "32"
	}

	if iconType == "" {
		iconType = "aim"
	}

	// for now, redirect to a placeholder icon
	iconURL := "/static/icons/default_" + iconType + "_" + size + ".png"
	// If it's an email lookup, extract username
	if strings.Contains(name, "@") {
		parts := strings.Split(name, "@")
		if len(parts) > 0 {
			name = parts[0]
		}
	}

	// check if user is online and get their state
	screenName := state.NewIdentScreenName(name)
	if session := h.SessionRetriever.RetrieveSession(screenName); session != nil {
		if session.Away() {
			iconURL = "/static/icons/away_" + iconType + "_" + size + ".png"
		} else if session.Idle() {
			iconURL = "/static/icons/idle_" + iconType + "_" + size + ".png"
		} else {
			iconURL = "/static/icons/online_" + iconType + "_" + size + ".png"
		}
	} else {
		iconURL = "/static/icons/offline_" + iconType + "_" + size + ".png"
	}

	// redirect to icon URL
	http.Redirect(w, r, iconURL, http.StatusFound)
}

// SetState handles GET /presence/setState requests to update user's presence state.
func (h *PresenceHandler) SetState(w http.ResponseWriter, r *http.Request) {
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

	// get the requested state
	stateParam := r.URL.Query().Get("state")
	awayMsg := r.URL.Query().Get("awayMsg")
	// get OSCAR session if available
	oscarSession := session.OSCARSession
	if oscarSession == nil {
		// fr web-only sessions, we'll need to track state in the WebAPI session
		// for now, just store in event data
		h.Logger.WarnContext(ctx, "no OSCAR session for presence update", "aimsid", aimsid)
		// still send success response
		response := BaseResponse{}
		response.Response.StatusCode = 200
		response.Response.StatusText = "OK"
		SendResponse(w, r, response, h.Logger)
		return
	}

	// map web state to OSCAR status bits
	var statusBitmask uint32
	switch stateParam {
	case "online":
		statusBitmask = 0x0000 // Clear all status bits
		oscarSession.SetAwayMessage("")
	case "away":
		statusBitmask = wire.OServiceUserStatusAway
		if awayMsg != "" {
			oscarSession.SetAwayMessage(awayMsg)
		}
	case "invisible":
		statusBitmask = wire.OServiceUserStatusInvisible
	case "dnd":
		statusBitmask = wire.OServiceUserStatusDND
	default:
		h.sendError(w, http.StatusBadRequest, "invalid state parameter")
		return
	}

	// update OSCAR session status
	oscarSession.SetUserStatusBitmask(statusBitmask)
	// broadcast presence update
	if statusBitmask&wire.OServiceUserStatusInvisible != 0 {
		// user going invisible
		// broadcast departure
		if err := h.BuddyBroadcaster.BroadcastBuddyDeparted(ctx, oscarSession); err != nil {
			h.Logger.ErrorContext(ctx, "failed to broadcast buddy departed", "err", err.Error())
		}
	} else {
		// user visible
		// broadcast arrival/update
		if err := h.BuddyBroadcaster.BroadcastBuddyArrived(ctx, oscarSession.IdentScreenName(), oscarSession.Session().TLVUserInfo()); err != nil {
			h.Logger.ErrorContext(ctx, "failed to broadcast buddy arrived", "err", err.Error())
		}
	}

	// queue presence event for other WebAPI sessions watching this user
	h.broadcastPresenceEvent(session.ScreenName.IdentScreenName(), stateParam, awayMsg, "")
	h.Logger.InfoContext(ctx, "presence state updated",
		"screenName", session.ScreenName.String(),
		"state", stateParam,
		"hasAwayMsg", awayMsg != "",
	)

	// send success response
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	SendResponse(w, r, response, h.Logger)
}

// SetStatus handles GET /presence/setStatus requests to update user's status message.
func (h *PresenceHandler) SetStatus(w http.ResponseWriter, r *http.Request) {
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

	// get the status message
	statusMsg := r.URL.Query().Get("statusMsg")
	statusCode := r.URL.Query().Get("statusCode")

	// store status message in session (this would normally be stored in a profile/status service)
	// for now, we'll broadcast it as part of presence
	// get OSCAR session if available
	if oscarSession := session.OSCARSession; oscarSession != nil {
		// in OSCAR, status messages are typically part of the profile
		// we'll need to extend this based on the actual implementation
		// broadcast presence update with new status
		if err := h.BuddyBroadcaster.BroadcastBuddyArrived(ctx, oscarSession.IdentScreenName(), oscarSession.Session().TLVUserInfo()); err != nil {
			h.Logger.ErrorContext(ctx, "failed to broadcast status update", "err", err.Error())
		}
	}

	// queue status event for other WebAPI sessions
	h.broadcastPresenceEvent(session.ScreenName.IdentScreenName(), "", "", statusMsg)
	h.Logger.InfoContext(ctx, "status message updated",
		"screenName", session.ScreenName.String(),
		"statusMsg", statusMsg,
		"statusCode", statusCode,
	)

	// send success response
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	SendResponse(w, r, response, h.Logger)
}

// SetProfile handles GET /presence/setProfile requests to update user's profile.
func (h *PresenceHandler) SetProfile(w http.ResponseWriter, r *http.Request) {
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

	// get the profile content
	profileText := r.URL.Query().Get("profile")
	// limit profile size (4KB max)
	if len(profileText) > 4096 {
		h.sendError(w, http.StatusBadRequest, "profile too large (max 4KB)")
		return
	}

	// save profile using ProfileManager
	profile := state.UserProfile{
		ProfileText: profileText,
		UpdateTime:  time.Now().UTC(),
	}
	if err := h.ProfileManager.SetProfile(ctx, session.ScreenName.IdentScreenName(), profile); err != nil {
		h.Logger.ErrorContext(ctx, "failed to set profile", "err", err.Error())
		h.sendError(w, http.StatusInternalServerError, "failed to save profile")
		return
	}

	h.Logger.InfoContext(ctx, "profile updated",
		"screenName", session.ScreenName.String(),
		"profileSize", len(profileText),
	)

	// send success response
	response := BaseResponse{}
	response.Response.StatusCode = 200
	response.Response.StatusText = "OK"
	SendResponse(w, r, response, h.Logger)
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

// getUserPresence gets the current presence state for a user.
func (h *PresenceHandler) getUserPresence(screenName state.IdentScreenName) BuddyPresenceInfo {
	// default offline presence
	presence := BuddyPresenceInfo{
		AimID:    screenName.String(),
		State:    "offline",
		UserType: "aim",
	}

	// check if user is online by looking for their OSCAR session
	if session := h.SessionRetriever.RetrieveSession(screenName); session != nil {
		presence.State = "online"
		// check user status
		if session.Away() {
			presence.State = "away"
		} else if session.AllUserStatusBitmask(wire.OServiceUserStatusDND) {
			presence.State = "dnd"
		}

		// check idle time
		if session.Idle() {
			presence.State = "idle"
			idleTime := time.Since(session.IdleTime())
			presence.IdleTime = int(idleTime.Minutes())
		}

		// get online time
		presence.OnlineTime = session.SignonTime().Unix()
	}

	// determine user type
	if strings.HasPrefix(screenName.String(), "admin") {
		presence.UserType = "admin"
	} else if isICQScreenName(screenName.String()) {
		presence.UserType = "icq"
	}

	return presence
}

// getBuddyListGroups retrieves the buddy list organized by groups.
func (h *PresenceHandler) getBuddyListGroups(ctx context.Context, screenName state.IdentScreenName) ([]BuddyGroupInfo, error) {
	// get feedbag items
	items, err := h.FeedbagRetriever.RetrieveFeedbag(ctx, screenName)
	if err != nil {
		return nil, err
	}

	// organize items into groups
	groupMap := make(map[uint16]*BuddyGroupInfo)
	buddyToGroup := make(map[string]uint16)
	// first pass: identify groups
	for _, item := range items {
		if item.ClassID == wire.FeedbagClassIdGroup {
			name := item.Name
			if name == "" {
				// default group name
				name = "Buddies"
			}

			groupMap[item.ItemID] = &BuddyGroupInfo{
				Name:    name,
				Buddies: []BuddyPresenceInfo{},
			}
		}
	}

	// second pass: add buddies to groups
	for _, item := range items {
		if item.ClassID == wire.FeedbagClassIdBuddy {
			// get buddy screen name
			buddyName := item.Name
			if buddyName == "" {
				continue
			}

			// find buddy's group
			groupID := item.GroupID
			buddyToGroup[buddyName] = groupID
		}
	}

	// if no groups exist, create a default one
	if len(groupMap) == 0 {
		groupMap[0] = &BuddyGroupInfo{
			Name:    "Buddies",
			Buddies: []BuddyPresenceInfo{},
		}
	}

	// add buddies to their groups with presence info
	for buddyName, groupID := range buddyToGroup {
		group, exists := groupMap[groupID]
		if !exists {
			// put in first available group if group doesn't exist
			for _, g := range groupMap {
				group = g
				break
			}
		}

		buddyScreenName := state.NewIdentScreenName(buddyName)
		// check blocking relationship (OSCAR compliant)
		rel, err := h.RelationshipFetcher.Relationship(ctx, screenName, buddyScreenName)
		if err != nil {
			h.Logger.WarnContext(ctx, "failed to get relationship", "error", err)
			// on error, include the buddy but they'll appear offline
			presence := BuddyPresenceInfo{
				AimID:    buddyName,
				State:    "offline",
				UserType: "aim",
			}
			group.Buddies = append(group.Buddies, presence)
			continue
		}

		// OSCAR compliance: mutual invisibility when blocking
		if rel.YouBlock || rel.BlocksYou {
			// add them as offline to maintain buddy list structure
			presence := BuddyPresenceInfo{
				AimID:    buddyName,
				State:    "offline",
				UserType: "aim",
			}
			group.Buddies = append(group.Buddies, presence)
		} else {
			// normal presence lookup
			presence := h.getUserPresence(buddyScreenName)
			group.Buddies = append(group.Buddies, presence)
		}
	}

	// convert map to slice
	groups := make([]BuddyGroupInfo, 0, len(groupMap))
	for _, group := range groupMap {
		groups = append(groups, *group)
	}

	return groups, nil
}

// isICQScreenName checks if a screen name is an ICQ number.
func isICQScreenName(screenName string) bool {
	if len(screenName) == 0 {
		return false
	}

	for _, r := range screenName {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
