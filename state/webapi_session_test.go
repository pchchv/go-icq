package state

import (
	"testing"
	"time"

	"github.com/pchchv/go-icq/server/webapi/types"
	"github.com/stretchr/testify/assert"
)

func TestWebAPISession_TempBuddies(t *testing.T) {
	tests := []struct {
		name           string
		setupSession   func() *WebAPISession
		operations     func(*WebAPISession)
		expectedChecks func(*testing.T, *WebAPISession)
	}{
		{
			name: "Initialize_NilTempBuddies",
			setupSession: func() *WebAPISession {
				return &WebAPISession{
					AimSID:       "test-session",
					ScreenName:   DisplayScreenName("testuser"),
					EventQueue:   types.NewEventQueue(100),
					CreatedAt:    time.Now(),
					LastAccessed: time.Now(),
					ExpiresAt:    time.Now().Add(time.Hour),
				}
			},
			operations: func(s *WebAPISession) {
				// initialize TempBuddies if nil
				if s.TempBuddies == nil {
					s.TempBuddies = make(map[string]bool)
				}
				s.TempBuddies["buddy1"] = true
			},
			expectedChecks: func(t *testing.T, s *WebAPISession) {
				assert.NotNil(t, s.TempBuddies)
				assert.True(t, s.TempBuddies["buddy1"])
				assert.Equal(t, 1, len(s.TempBuddies))
			},
		},
		{
			name: "Add_MultipleTempBuddies",
			setupSession: func() *WebAPISession {
				return &WebAPISession{
					AimSID:       "test-session",
					ScreenName:   DisplayScreenName("testuser"),
					TempBuddies:  make(map[string]bool),
					EventQueue:   types.NewEventQueue(100),
					CreatedAt:    time.Now(),
					LastAccessed: time.Now(),
					ExpiresAt:    time.Now().Add(time.Hour),
				}
			},
			operations: func(s *WebAPISession) {
				s.TempBuddies["buddy1"] = true
				s.TempBuddies["buddy2"] = true
				s.TempBuddies["buddy3"] = true
			},
			expectedChecks: func(t *testing.T, s *WebAPISession) {
				assert.Equal(t, 3, len(s.TempBuddies))
				assert.True(t, s.TempBuddies["buddy1"])
				assert.True(t, s.TempBuddies["buddy2"])
				assert.True(t, s.TempBuddies["buddy3"])
			},
		},
		{
			name: "Add_DuplicateTempBuddy",
			setupSession: func() *WebAPISession {
				return &WebAPISession{
					AimSID:       "test-session",
					ScreenName:   DisplayScreenName("testuser"),
					TempBuddies:  map[string]bool{"buddy1": true},
					EventQueue:   types.NewEventQueue(100),
					CreatedAt:    time.Now(),
					LastAccessed: time.Now(),
					ExpiresAt:    time.Now().Add(time.Hour),
				}
			},
			operations: func(s *WebAPISession) {
				// add the same buddy again
				s.TempBuddies["buddy1"] = true
			},
			expectedChecks: func(t *testing.T, s *WebAPISession) {
				// should still only have one entry
				assert.Equal(t, 1, len(s.TempBuddies))
				assert.True(t, s.TempBuddies["buddy1"])
			},
		},
		{
			name: "Remove_TempBuddy",
			setupSession: func() *WebAPISession {
				return &WebAPISession{
					AimSID:     "test-session",
					ScreenName: DisplayScreenName("testuser"),
					TempBuddies: map[string]bool{
						"buddy1": true,
						"buddy2": true,
					},
					EventQueue:   types.NewEventQueue(100),
					CreatedAt:    time.Now(),
					LastAccessed: time.Now(),
					ExpiresAt:    time.Now().Add(time.Hour),
				}
			},
			operations: func(s *WebAPISession) {
				delete(s.TempBuddies, "buddy1")
			},
			expectedChecks: func(t *testing.T, s *WebAPISession) {
				assert.Equal(t, 1, len(s.TempBuddies))
				assert.False(t, s.TempBuddies["buddy1"])
				assert.True(t, s.TempBuddies["buddy2"])
			},
		},
		{
			name: "Check_NonExistentBuddy",
			setupSession: func() *WebAPISession {
				return &WebAPISession{
					AimSID:       "test-session",
					ScreenName:   DisplayScreenName("testuser"),
					TempBuddies:  map[string]bool{"buddy1": true},
					EventQueue:   types.NewEventQueue(100),
					CreatedAt:    time.Now(),
					LastAccessed: time.Now(),
					ExpiresAt:    time.Now().Add(time.Hour),
				}
			},
			operations: func(s *WebAPISession) {
				// no operations, just checking
			},
			expectedChecks: func(t *testing.T, s *WebAPISession) {
				assert.False(t, s.TempBuddies["nonexistent"])
				assert.True(t, s.TempBuddies["buddy1"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup
			session := tt.setupSession()
			// perform operations
			tt.operations(session)
			// verify
			tt.expectedChecks(t, session)
		})
	}
}

func TestWebAPISession_TempBuddiesIndependence(t *testing.T) {
	// test that temp buddies are independent across sessions
	session1 := &WebAPISession{
		AimSID:      "session1",
		ScreenName:  DisplayScreenName("user1"),
		TempBuddies: map[string]bool{"buddy1": true},
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	session2 := &WebAPISession{
		AimSID:      "session2",
		ScreenName:  DisplayScreenName("user2"),
		TempBuddies: map[string]bool{"buddy2": true},
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	// verify sessions have independent temp buddies
	assert.True(t, session1.TempBuddies["buddy1"])
	assert.False(t, session1.TempBuddies["buddy2"])

	assert.False(t, session2.TempBuddies["buddy1"])
	assert.True(t, session2.TempBuddies["buddy2"])

	// modify one session's temp buddies
	session1.TempBuddies["buddy3"] = true

	// verify it doesn't affect the other session
	assert.True(t, session1.TempBuddies["buddy3"])
	assert.False(t, session2.TempBuddies["buddy3"])
}
