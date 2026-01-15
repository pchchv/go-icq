package types

const (
	// Event types that can be subscribed to:
	EventTypeBuddyList    EventType = "buddylist"
	EventTypePresence     EventType = "presence"
	EventTypeIM           EventType = "im"
	EventTypeSentIM       EventType = "sentIM"
	EventTypeTyping       EventType = "typing"
	EventTypeStatus       EventType = "status"
	EventTypeOfflineIM    EventType = "offlineIM"
	EventTypeSessionEnded EventType = "sessionEnded"
	EventTypeRateLimit    EventType = "rateLimit"
)

// UserInfo represents basic user information in events.
type UserInfo struct {
	AimID      string  `json:"aimId"`
	DisplayID  string  `json:"displayId,omitempty"`
	UserType   string  `json:"userType,omitempty"`
	State      string  `json:"state,omitempty"`
	OnlineTime float64 `json:"onlineTime,omitempty"` // float64 for AMF3 encoding
}

// EventType defines the type of WebAPI event.
type EventType string

// IMEvent represents an instant message event.
type IMEvent struct {
	From      string  `json:"from"`
	Message   string  `json:"message"`
	Timestamp float64 `json:"timestamp"` // float64 for AMF3 encoding
	AutoResp  bool    `json:"autoResponse,omitempty"`
}

// SentIMEvent represents a sent instant message event.
type SentIMEvent struct {
	Sender    UserInfo `json:"sender"` // Sender user info
	Dest      UserInfo `json:"dest"`   // Destination user info
	Message   string   `json:"message"`
	Timestamp float64  `json:"timestamp"` // float64 for AMF3 encoding
	AutoResp  bool     `json:"autoResponse,omitempty"`
}

// TypingEvent represents a typing notification event.
type TypingEvent struct {
	From   string `json:"from"`
	Typing bool   `json:"typing"`
}

// BuddyListEvent represents a buddy list change event.
type BuddyListEvent struct {
	Action string      `json:"action"` // "add", "remove", "update"
	Group  string      `json:"group,omitempty"`
	Buddy  interface{} `json:"buddy"`
}

// PresenceEvent represents a presence change event.
type PresenceEvent struct {
	AimID      string `json:"aimId"`
	State      string `json:"state"` // "online", "offline", "away", "idle"
	StatusMsg  string `json:"statusMsg,omitempty"`
	AwayMsg    string `json:"awayMsg,omitempty"`
	IdleTime   int    `json:"idleTime,omitempty"`   // Minutes idle
	OnlineTime int64  `json:"onlineTime,omitempty"` // Unix timestamp
	UserType   string `json:"userType"`             // "aim", "icq", "admin"
}
