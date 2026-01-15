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

// EventType defines the type of WebAPI event.
type EventType string
