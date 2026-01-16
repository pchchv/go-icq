package state

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"
)

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

// ChatUserEventData represents user join/leave event data.
type ChatUserEventData struct {
	ScreenName string `json:"screenName"`
	Timestamp  int64  `json:"timestamp"`
}

// ChatTypingEventData represents typing status event data.
type ChatTypingEventData struct {
	ScreenName   string `json:"screenName"`
	TypingStatus string `json:"typingStatus"`
}

// ChatParticipant represents a participant in a chat room.
type ChatParticipant struct {
	RoomID          string
	ChatSID         string
	JoinedAt        int64
	ScreenName      string
	TypingStatus    string
	TypingUpdatedAt *int64
}

// ChatParticipantList represents a list of participants in the room.
type ChatParticipantList struct {
	Participants []string `json:"participants"`
}

// WebAPIChatRoom represents a chat room for Web API.
type WebAPIChatRoom struct {
	RoomID            string       `json:"roomId"`
	RoomName          string       `json:"roomName"`
	Description       string       `json:"description,omitempty"`
	RoomType          ChatRoomType `json:"roomType"`
	CategoryID        string       `json:"categoryId,omitempty"`
	CreatorScreenName string       `json:"-"` // Internal only
	CreatedAt         int64        `json:"-"`
	ClosedAt          *int64       `json:"-"`
	MaxParticipants   int          `json:"-"`
	InstanceID        int          `json:"instanceId"`
}

// WebAPIChatManager manages Web API chat rooms.
type WebAPIChatManager struct {
	store    *SQLiteUserStore
	logger   *slog.Logger
	sessions *WebAPISessionManager
	mu       sync.RWMutex
	// In-memory cache for active rooms
	activeRooms map[string]*WebAPIChatRoom
	// Track typing timeouts
	typingTimers map[string]*time.Timer
}

// NewWebAPIChatManager creates a new WebAPIChatManager
func (s *SQLiteUserStore) NewWebAPIChatManager(logger *slog.Logger, sessions *WebAPISessionManager) *WebAPIChatManager {
	return &WebAPIChatManager{
		store:        s,
		logger:       logger,
		sessions:     sessions,
		activeRooms:  make(map[string]*WebAPIChatRoom),
		typingTimers: make(map[string]*time.Timer),
	}
}

func (m *WebAPIChatManager) generateInstanceID() int {
	return int(time.Now().Unix() % 1000000)
}

func (m *WebAPIChatManager) generateRoomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *WebAPIChatManager) generateChatSID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *WebAPIChatManager) getRoomByID(ctx context.Context, roomID string) (*WebAPIChatRoom, error) {
	var room WebAPIChatRoom
	err := m.store.db.QueryRowContext(ctx, `
		SELECT room_id, room_name, description, room_type, category_id,
		       creator_screen_name, created_at, closed_at, max_participants
		FROM web_chat_rooms
		WHERE room_id = ? AND closed_at IS NULL`,
		roomID).Scan(
		&room.RoomID, &room.RoomName, &room.Description, &room.RoomType,
		&room.CategoryID, &room.CreatorScreenName, &room.CreatedAt,
		&room.ClosedAt, &room.MaxParticipants)
	if err != nil {
		return nil, err
	}

	room.InstanceID = m.generateInstanceID()
	return &room, nil
}

func (m *WebAPIChatManager) getRoomByName(ctx context.Context, roomName string) (*WebAPIChatRoom, error) {
	var room WebAPIChatRoom
	err := m.store.db.QueryRowContext(ctx, `
		SELECT room_id, room_name, description, room_type, category_id,
		       creator_screen_name, created_at, closed_at, max_participants
		FROM web_chat_rooms
		WHERE room_name = ? AND closed_at IS NULL`,
		roomName).Scan(
		&room.RoomID, &room.RoomName, &room.Description, &room.RoomType,
		&room.CategoryID, &room.CreatorScreenName, &room.CreatedAt,
		&room.ClosedAt, &room.MaxParticipants)
	if err != nil {
		return nil, err
	}

	room.InstanceID = m.generateInstanceID()
	return &room, nil
}

func (m *WebAPIChatManager) getUserSessionInRoom(ctx context.Context, aimsid, roomID string) (*ChatSession, error) {
	var session ChatSession
	err := m.store.db.QueryRowContext(ctx, `
		SELECT chat_sid, aimsid, room_id, screen_name, instance_id, joined_at, left_at
		FROM web_chat_sessions
		WHERE aimsid = ? AND room_id = ? AND left_at IS NULL`,
		aimsid, roomID).Scan(
		&session.ChatSID, &session.AIMSid, &session.RoomID,
		&session.ScreenName, &session.InstanceID, &session.JoinedAt, &session.LeftAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (m *WebAPIChatManager) getUserSessionInRoomByScreenName(ctx context.Context, roomID, screenName string) (*ChatSession, error) {
	var session ChatSession
	err := m.store.db.QueryRowContext(ctx, `
		SELECT chat_sid, aimsid, room_id, screen_name, instance_id, joined_at, left_at
		FROM web_chat_sessions
		WHERE room_id = ? AND screen_name = ? AND left_at IS NULL`,
		roomID, screenName).Scan(
		&session.ChatSID, &session.AIMSid, &session.RoomID,
		&session.ScreenName, &session.InstanceID, &session.JoinedAt, &session.LeftAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}
