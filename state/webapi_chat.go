package state

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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

// NewWebAPIChatManager creates a new WebAPIChatManager.
func (s *SQLiteUserStore) NewWebAPIChatManager(logger *slog.Logger, sessions *WebAPISessionManager) *WebAPIChatManager {
	return &WebAPIChatManager{
		store:        s,
		logger:       logger,
		sessions:     sessions,
		activeRooms:  make(map[string]*WebAPIChatRoom),
		typingTimers: make(map[string]*time.Timer),
	}
}

// GetRecentMessages returns recent messages from a chat room (for history).
func (m *WebAPIChatManager) GetRecentMessages(ctx context.Context, roomID string, limit int) ([]*ChatMessage, error) {
	rows, err := m.store.db.QueryContext(ctx, `
		SELECT id, room_id, screen_name, message, whisper_target, timestamp
		FROM web_chat_messages
		WHERE room_id = ?
		ORDER BY timestamp DESC
		LIMIT ?`,
		roomID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*ChatMessage
	for rows.Next() {
		var msg ChatMessage
		err := rows.Scan(&msg.ID, &msg.RoomID, &msg.ScreenName, &msg.Message, &msg.WhisperTarget, &msg.Timestamp)
		if err == nil {
			messages = append(messages, &msg)
		}
	}

	// reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SendMessage sends a message to a chat room.
func (m *WebAPIChatManager) SendMessage(ctx context.Context, chatsid, message, whisperTarget string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// get session
	session, err := m.getSessionByChatSID(ctx, chatsid)
	if err != nil {
		return fmt.Errorf("invalid chat session: %w", err)
	}

	// verify user is still in room
	if session.LeftAt != nil {
		return errors.New("user has left the chat room")
	}

	// store message in database
	timestamp := time.Now().Unix()
	_, err = m.store.db.ExecContext(ctx, `
		INSERT INTO web_chat_messages (room_id, screen_name, message, whisper_target, timestamp)
		VALUES (?, ?, ?, ?, ?)`,
		session.RoomID, session.ScreenName, message, whisperTarget, timestamp)
	if err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	// broadcast message event
	eventData := ChatMessageEventData{
		ScreenName:    session.ScreenName,
		Message:       message,
		Timestamp:     timestamp,
		WhisperTarget: whisperTarget,
	}

	if whisperTarget != "" {
		// for whispers, only send to sender and target
		m.sendChatEventToUser(session.AIMSid, ChatEventData{
			ChatSID:   chatsid,
			EventType: ChatEventMessage,
			EventData: eventData,
		})
		// find target's session and send to them
		targetSession, _ := m.getUserSessionInRoomByScreenName(ctx, session.RoomID, whisperTarget)
		if targetSession != nil {
			m.sendChatEventToUser(targetSession.AIMSid, ChatEventData{
				ChatSID:   targetSession.ChatSID,
				EventType: ChatEventMessage,
				EventData: eventData,
			})
		}
	} else {
		// broadcast to all participants
		m.broadcastChatEvent(session.RoomID, ChatEventData{
			ChatSID:   chatsid,
			EventType: ChatEventMessage,
			EventData: eventData,
		})
	}

	return nil
}

// SetTyping sets the typing status for a user in a chat room.
func (m *WebAPIChatManager) SetTyping(ctx context.Context, chatsid, typingStatus string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// get session
	session, err := m.getSessionByChatSID(ctx, chatsid)
	if err != nil {
		return fmt.Errorf("invalid chat session: %w", err)
	}

	// verify user is still in room
	if session.LeftAt != nil {
		return errors.New("user has left the chat room")
	}

	// update typing status
	now := time.Now().Unix()
	_, err = m.store.db.ExecContext(ctx, `
		UPDATE web_chat_participants
		SET typing_status = ?, typing_updated_at = ?
		WHERE room_id = ? AND screen_name = ?`,
		typingStatus, now, session.RoomID, session.ScreenName)
	if err != nil {
		return fmt.Errorf("failed to update typing status: %w", err)
	}

	// cancel existing typing timer for this user
	timerKey := fmt.Sprintf("%s:%s", session.RoomID, session.ScreenName)
	if timer, exists := m.typingTimers[timerKey]; exists {
		timer.Stop()
		delete(m.typingTimers, timerKey)
	}

	// if status is "typing" or "typed", set a timer to reset it
	if typingStatus == "typing" || typingStatus == "typed" {
		timer := time.AfterFunc(10*time.Second, func() {
			m.mu.Lock()
			defer m.mu.Unlock()
			// reset typing status to none using background context here since this is
			// an async timer callback and the original context may have expired
			m.store.db.ExecContext(context.Background(), `
				UPDATE web_chat_participants
				SET typing_status = 'none', typing_updated_at = ?
				WHERE room_id = ? AND screen_name = ?`,
				time.Now().Unix(), session.RoomID, session.ScreenName)
			// broadcast the reset
			m.broadcastChatEvent(session.RoomID, ChatEventData{
				ChatSID:   chatsid,
				EventType: ChatEventTyping,
				EventData: ChatTypingEventData{
					ScreenName:   session.ScreenName,
					TypingStatus: "none",
				},
			})
			delete(m.typingTimers, timerKey)
		})
		m.typingTimers[timerKey] = timer
	}

	// broadcast typing event
	m.broadcastChatEvent(session.RoomID, ChatEventData{
		ChatSID:   chatsid,
		EventType: ChatEventTyping,
		EventData: ChatTypingEventData{
			ScreenName:   session.ScreenName,
			TypingStatus: typingStatus,
		},
	})

	return nil
}

// LeaveChat removes a user from a chat room.
func (m *WebAPIChatManager) LeaveChat(ctx context.Context, chatsid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// get session
	session, err := m.getSessionByChatSID(ctx, chatsid)
	if err != nil {
		return fmt.Errorf("invalid chat session: %w", err)
	}

	// mark session as left
	now := time.Now().Unix()
	_, err = m.store.db.ExecContext(ctx, `
		UPDATE web_chat_sessions
		SET left_at = ?
		WHERE chat_sid = ?`,
		now, chatsid)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// remove from participants
	_, err = m.store.db.ExecContext(ctx, `
		DELETE FROM web_chat_participants
		WHERE room_id = ? AND screen_name = ?`,
		session.RoomID, session.ScreenName)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	// cancel any typing timer
	timerKey := fmt.Sprintf("%s:%s", session.RoomID, session.ScreenName)
	if timer, exists := m.typingTimers[timerKey]; exists {
		timer.Stop()
		delete(m.typingTimers, timerKey)
	}

	// broadcast user left event
	// NOTE: Broadcasting doesn't need context as it's fire-and-forget
	m.broadcastChatEvent(session.RoomID, ChatEventData{
		ChatSID:   chatsid,
		EventType: ChatEventUserLeft,
		EventData: ChatUserEventData{
			ScreenName: session.ScreenName,
			Timestamp:  now,
		},
	})

	// check if room should be closed (no participants left)
	count, _ := m.getParticipantCount(ctx, session.RoomID)
	if count == 0 {
		m.closeRoom(ctx, session.RoomID)
	}

	return nil
}

// CleanupInactiveSessions removes sessions that have been inactive for too long.
func (m *WebAPIChatManager) CleanupInactiveSessions(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// mark sessions as left if they've been inactive for more than 30 minutes
	cutoff := time.Now().Add(-30 * time.Minute).Unix()
	rows, err := m.store.db.QueryContext(ctx, `
		SELECT chat_sid, room_id, screen_name
		FROM web_chat_sessions
		WHERE left_at IS NULL AND joined_at < ?`,
		cutoff)
	if err != nil {
		m.logger.Error("failed to get inactive sessions", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var chatsid, roomID, screenName string
		if err := rows.Scan(&chatsid, &roomID, &screenName); err != nil {
			continue
		}

		// mark as left
		now := time.Now().Unix()
		m.store.db.ExecContext(ctx, `UPDATE web_chat_sessions SET left_at = ? WHERE chat_sid = ?`, now, chatsid)
		m.store.db.ExecContext(ctx, `DELETE FROM web_chat_participants WHERE room_id = ? AND screen_name = ?`,
			roomID, screenName)

		// broadcast user left
		// NOTE: Broadcasting doesn't need context as it's fire-and-forget
		m.broadcastChatEvent(roomID, ChatEventData{
			ChatSID:   chatsid,
			EventType: ChatEventUserLeft,
			EventData: ChatUserEventData{
				ScreenName: screenName,
				Timestamp:  now,
			},
		})
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

func (m *WebAPIChatManager) getParticipants(ctx context.Context, roomID string) ([]string, error) {
	rows, err := m.store.db.QueryContext(ctx, `
		SELECT screen_name FROM web_chat_participants WHERE room_id = ?`,
		roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []string
	for rows.Next() {
		var screenName string
		if err := rows.Scan(&screenName); err != nil {
			continue
		}
		participants = append(participants, screenName)
	}

	return participants, nil
}

func (m *WebAPIChatManager) getParticipantCount(ctx context.Context, roomID string) (int, error) {
	var count int
	err := m.store.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM web_chat_participants WHERE room_id = ?`,
		roomID).Scan(&count)
	return count, err
}

func (m *WebAPIChatManager) getSessionByChatSID(ctx context.Context, chatsid string) (*ChatSession, error) {
	var session ChatSession
	err := m.store.db.QueryRowContext(ctx, `
		SELECT chat_sid, aimsid, room_id, screen_name, instance_id, joined_at, left_at
		FROM web_chat_sessions
		WHERE chat_sid = ?`,
		chatsid).Scan(
		&session.ChatSID, &session.AIMSid, &session.RoomID,
		&session.ScreenName, &session.InstanceID, &session.JoinedAt, &session.LeftAt)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (m *WebAPIChatManager) sendChatEventToUser(aimsid string, event ChatEventData) {
	// get the user's Web API session using background context for async event sending
	session, err := m.sessions.GetSession(context.Background(), aimsid)
	if err != nil {
		m.logger.Error("failed to get session for chat event", "error", err, "aimsid", aimsid)
		return
	}

	// queue the chat event
	session.EventQueue.Push("chat", event)
}

func (m *WebAPIChatManager) broadcastChatEvent(roomID string, event ChatEventData) {
	// Get all active sessions in the room
	rows, err := m.store.db.Query(`
		SELECT aimsid, chat_sid FROM web_chat_sessions
		WHERE room_id = ? AND left_at IS NULL`,
		roomID)
	if err != nil {
		m.logger.Error("failed to get sessions for broadcast", "error", err, "roomID", roomID)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var aimsid, chatsid string
		if err := rows.Scan(&aimsid, &chatsid); err != nil {
			continue
		}

		// update event with the recipient's chat session ID if not set
		if event.ChatSID == "" {
			event.ChatSID = chatsid
		}

		m.sendChatEventToUser(aimsid, event)
	}
}

func (m *WebAPIChatManager) createRoom(ctx context.Context, roomName, creatorScreenName string) (*WebAPIChatRoom, error) {
	room := &WebAPIChatRoom{
		RoomID:            m.generateRoomID(),
		RoomName:          roomName,
		Description:       fmt.Sprintf("Chat room created by %s", creatorScreenName),
		RoomType:          ChatRoomTypeUserCreated,
		CreatorScreenName: creatorScreenName,
		CreatedAt:         time.Now().Unix(),
		MaxParticipants:   100,
		InstanceID:        m.generateInstanceID(),
	}

	_, err := m.store.db.ExecContext(ctx, `
		INSERT INTO web_chat_rooms (room_id, room_name, description, room_type,
		                            category_id, creator_screen_name, created_at, max_participants)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		room.RoomID, room.RoomName, room.Description, room.RoomType,
		room.CategoryID, room.CreatorScreenName, room.CreatedAt, room.MaxParticipants)
	if err != nil {
		return nil, err
	}

	// cache the room
	m.activeRooms[room.RoomID] = room
	return room, nil
}

func (m *WebAPIChatManager) closeRoom(ctx context.Context, roomID string) {
	now := time.Now().Unix()
	m.store.db.ExecContext(ctx, `
		UPDATE web_chat_rooms SET closed_at = ? WHERE room_id = ?`,
		now, roomID)

	// remove from cache
	delete(m.activeRooms, roomID)

	// broadcast room closed event
	// NOTE: Broadcasting doesn't need context as it's fire-and-forget
	m.broadcastChatEvent(roomID, ChatEventData{
		EventType: ChatEventClosed,
	})
}
