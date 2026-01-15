package types

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

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

// Event represents an event to be delivered to a web client.
type Event struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	SeqNum    uint64      `json:"seqNum"`
	Timestamp int64       `json:"timestamp"`
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

// EventQueue manages a queue of events for a WebAPI session.
type EventQueue struct {
	events   []Event
	seqNum   uint64
	maxSize  int
	mu       sync.RWMutex
	waitChan chan struct{}
	closed   bool
	closedMu sync.RWMutex
}

// NewEventQueue creates a new event queue with the specified maximum size.
func NewEventQueue(maxSize int) *EventQueue {
	return &EventQueue{
		events:   make([]Event, 0),
		maxSize:  maxSize,
		waitChan: make(chan struct{}, 1),
	}
}

// Push adds an event to the queue.
func (q *EventQueue) Push(eventType EventType, data interface{}) {
	q.closedMu.RLock()
	if q.closed {
		q.closedMu.RUnlock()
		return
	}
	q.closedMu.RUnlock()

	q.mu.Lock()
	defer q.mu.Unlock()

	// increment sequence number atomically
	seqNum := atomic.AddUint64(&q.seqNum, 1)
	event := Event{
		Type:      eventType,
		SeqNum:    seqNum,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	// add event to queue
	q.events = append(q.events, event)

	// if queue exceeds max size, remove oldest events
	if len(q.events) > q.maxSize {
		// keep only the most recent maxSize events
		q.events = q.events[len(q.events)-q.maxSize:]
	}

	// signal any waiting fetchers
	select {
	case q.waitChan <- struct{}{}:
	default:
		// channel already has a signal
	}
}

// Size returns the current number of events in the queue.
func (q *EventQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.events)
}

// Close closes the event queue, unblocking any waiting fetchers.
func (q *EventQueue) Close() {
	q.closedMu.Lock()
	defer q.closedMu.Unlock()

	if q.closed {
		return
	}

	q.closed = true

	// send multiple signals to unblock all potential waiters
loop:
	for i := 0; i < 10; i++ {
		select {
		case q.waitChan <- struct{}{}:
		default:
			break loop
		}
	}
}

// IsClosed returns whether the queue is closed.
func (q *EventQueue) IsClosed() bool {
	q.closedMu.RLock()
	defer q.closedMu.RUnlock()

	return q.closed
}

// GetAllEvents returns all events in the queue (for debugging).
func (q *EventQueue) GetAllEvents() []Event {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]Event, len(q.events))
	copy(result, q.events)
	return result
}

// Clear removes all events from the queue.
func (q *EventQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.events = make([]Event, 0)
}

// Fetch retrieves events from the queue, optionally waiting for new events.
func (q *EventQueue) Fetch(ctx context.Context, lastSeqNum uint64, timeout time.Duration) ([]Event, error) {
	q.closedMu.RLock()
	if q.closed {
		q.closedMu.RUnlock()
		return nil, errors.New("event queue is closed")
	}
	q.closedMu.RUnlock()

	// first, check if we have any events newer than lastSeqNum
	q.mu.RLock()
	events := q.getEventsAfter(lastSeqNum)
	q.mu.RUnlock()

	if len(events) > 0 {
		return events, nil
	}

	// no events available, wait for new ones or timeout
	timeoutChan := time.After(timeout)
	for {
		select {
		case <-q.waitChan:
			// new events may be available
			q.mu.RLock()
			events = q.getEventsAfter(lastSeqNum)
			q.mu.RUnlock()

			if len(events) > 0 {
				return events, nil
			}
			// false alarm, keep waiting
		case <-timeoutChan:
			// timeout reached, return empty array
			return []Event{}, nil
		case <-ctx.Done():
			// context cancelled
			return nil, ctx.Err()
		}
	}
}

// getEventsAfter returns all events with sequence number greater than the specified value.
// Must be called with at least a read lock held.
func (q *EventQueue) getEventsAfter(seqNum uint64) (result []Event) {
	for _, event := range q.events {
		if event.SeqNum > seqNum {
			result = append(result, event)
		}
	}
	return
}
