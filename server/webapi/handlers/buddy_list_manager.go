package handlers

import "log/slog"

// WebAPIBuddyInfo represents a buddy in the WebAPI format.
type WebAPIBuddyInfo struct {
	AimID        string   `json:"aimId"`
	DisplayID    string   `json:"displayId"`
	State        string   `json:"state"` // "online", "offline", "away", "idle"
	StatusMsg    string   `json:"statusMsg,omitempty"`
	AwayMsg      string   `json:"awayMsg,omitempty"`
	OnlineTime   int64    `json:"onlineTime,omitempty"`
	IdleTime     int      `json:"idleTime,omitempty"` // Minutes idle
	UserType     string   `json:"userType"`           // "aim", "icq", "admin"
	Bot          bool     `json:"bot"`
	Service      string   `json:"service,omitempty"` // "aim", "icq"
	PresenceIcon string   `json:"presenceIcon,omitempty"`
	BuddyIcon    string   `json:"buddyIcon,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	MemberSince  int64    `json:"memberSince,omitempty"`
}

// WebAPIBuddyGroup represents a group in the WebAPI buddy list format.
type WebAPIBuddyGroup struct {
	Name    string            `json:"name"`
	Buddies []WebAPIBuddyInfo `json:"buddies"`
	Recent  bool              `json:"recent,omitempty"`
	Smart   interface{}       `json:"smart,omitempty"` // Can be null or number
}

// BuddyListManager handles the conversion of OSCAR feedbag data
// to WebAPI buddy list format for web clients.
type BuddyListManager struct {
	feedbagRetriever FeedbagRetriever
	sessionRetriever SessionRetriever
	logger           *slog.Logger
}

// NewBuddyListManager creates a new instance of the buddy list manager.
func NewBuddyListManager(feedbagRetriever FeedbagRetriever, sessionRetriever SessionRetriever, logger *slog.Logger) *BuddyListManager {
	return &BuddyListManager{
		feedbagRetriever: feedbagRetriever,
		sessionRetriever: sessionRetriever,
		logger:           logger,
	}
}

// FormatBuddyListEvent formats a buddy list for an event.
func (m *BuddyListManager) FormatBuddyListEvent(groups []WebAPIBuddyGroup) map[string]interface{} {
	// convert groups to a format that AMF3 can properly encode AMF3 has trouble with complex struct slices, convert to maps
	groupMaps := make([]interface{}, len(groups))
	for i, group := range groups {
		buddyMaps := make([]interface{}, len(group.Buddies))
		for j, buddy := range group.Buddies {
			// convert each buddy to a map
			buddyMap := map[string]interface{}{
				"aimId":     buddy.AimID,
				"displayId": buddy.DisplayID,
				"state":     buddy.State,
				"userType":  buddy.UserType,
				"bot":       buddy.Bot,
				"service":   buddy.Service,
			}

			// add optional fields if present
			if buddy.StatusMsg != "" {
				buddyMap["statusMsg"] = buddy.StatusMsg
			}

			if buddy.AwayMsg != "" {
				buddyMap["awayMsg"] = buddy.AwayMsg
			}

			if buddy.OnlineTime > 0 {
				buddyMap["onlineTime"] = float64(buddy.OnlineTime)
			}

			if buddy.IdleTime > 0 {
				buddyMap["idleTime"] = buddy.IdleTime
			}

			if buddy.PresenceIcon != "" {
				buddyMap["presenceIcon"] = buddy.PresenceIcon
			}

			if buddy.BuddyIcon != "" {
				buddyMap["buddyIcon"] = buddy.BuddyIcon
			}

			if len(buddy.Capabilities) > 0 {
				buddyMap["capabilities"] = buddy.Capabilities
			}

			if buddy.MemberSince > 0 {
				buddyMap["memberSince"] = float64(buddy.MemberSince)
			}

			buddyMaps[j] = buddyMap
		}

		// convert group to a map
		groupMap := map[string]interface{}{
			"name":    group.Name,
			"buddies": buddyMaps,
		}

		// add optional group fields
		if group.Recent {
			groupMap["recent"] = group.Recent
		}

		if group.Smart != nil {
			groupMap["smart"] = group.Smart
		}

		groupMaps[i] = groupMap
	}

	return map[string]interface{}{
		"groups": groupMaps,
	}
}

// GetPresenceForBuddy retrieves presence information for a specific buddy.
func (m *BuddyListManager) GetPresenceForBuddy(screenName string) WebAPIBuddyInfo {
	return m.getBuddyInfo(screenName)
}

// getBuddyInfo retrieves the current presence information for a buddy.
func (m *BuddyListManager) getBuddyInfo(buddyName string) WebAPIBuddyInfo {
	// default to offline
	info := WebAPIBuddyInfo{
		AimID:     buddyName,
		DisplayID: buddyName,
		State:     "offline",
		UserType:  "aim",
		Bot:       false,
		Service:   "aim",
	}

	// check if buddy is online
	buddyScreenName := state.NewIdentScreenName(buddyName)
	if session := m.sessionRetriever.RetrieveSession(buddyScreenName); session != nil {
		// buddy is online
		info.State = "online"
		info.OnlineTime = session.SignonTime().Unix()

		// check away status
		if session.Away() {
			info.State = "away"
			info.AwayMsg = session.AwayMessage()
		}

		// check idle status
		if session.Idle() {
			idleDuration := time.Since(session.IdleTime())
			info.IdleTime = int(idleDuration.Minutes())
			if info.State == "online" {
				info.State = "idle"
			}
		}

		// status messages not currently supported in SessionInstance
		// set capabilities
		// capabilities parsing not implemented
		info.Capabilities = []string{}
	}

	return info
}
