package state

const (
	ChatRoomTypeUserCreated ChatRoomType  = "userCreated"
	ChatEventUserEntered    ChatEventType = "userEntered"
	ChatEventUserInRoom     ChatEventType = "userInRoom"
	ChatEventUserLeft       ChatEventType = "userLeft"
	ChatEventMessage        ChatEventType = "message"
	ChatEventTyping         ChatEventType = "typing"
	ChatEventClosed         ChatEventType = "closed"
)

// ChatRoomType represents the type of chat room.
type ChatRoomType string

// ChatEventType represents the type of chat event.
type ChatEventType string

// ChatEventData represents data for a chat event.
type ChatEventData struct {
	ChatSID   string        `json:"chatsid"`
	EventType ChatEventType `json:"eventType"`
	EventData interface{}   `json:"eventData"`
}

// ChatSession represents a user's session in a chat room.
type ChatSession struct {
	ChatSID    string
	AIMSid     string
	RoomID     string
	ScreenName string
	InstanceID int
	JoinedAt   int64
	LeftAt     *int64
}

// ChatMessage represents a message sent in a chat room.
type ChatMessage struct {
	ID            int64
	RoomID        string
	Message       string
	Timestamp     int64
	ScreenName    string
	WhisperTarget string
}

// ChatMessageEventData represents chat message event data.
type ChatMessageEventData struct {
	ScreenName    string `json:"screenName"`
	Message       string `json:"message"`
	Timestamp     int64  `json:"timestamp"`
	WhisperTarget string `json:"whisperTarget,omitempty"`
}
