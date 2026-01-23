package handlers


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
