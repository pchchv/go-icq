package state

import (
	"context"
	"log/slog"
	"testing"

	"github.com/pchchv/go-icq/wire"
	"github.com/stretchr/testify/assert"
)

func TestInMemorySessionManager_AddSession_Timeout(t *testing.T) {
	sm := NewInMemorySessionManager(slog.Default())
	ctx, cancel := context.WithCancel(context.Background())
	sess1, err := sm.AddSession(ctx, "user-screen-name")
	assert.NoError(t, err)
	sess1.SetSignonComplete()

	go func() {
		<-sess1.Closed()
		cancel()
	}()

	sess2, err := sm.AddSession(ctx, "user-screen-name")
	assert.Nil(t, sess2)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestInMemorySessionManager_AddSession_SessionConflict(t *testing.T) {
	sm := NewInMemorySessionManager(slog.Default())
	ctx := context.Background()
	sess1, err := sm.AddSession(ctx, "user-screen-name")
	assert.NoError(t, err)
	sess1.SetSignonComplete()

	go func() {
		<-sess1.Closed()
		rec, ok := sm.store[NewIdentScreenName("user-screen-name")]
		if assert.True(t, ok) {
			close(rec.removed)
		}
	}()

	sess2, err := sm.AddSession(ctx, "user-screen-name")
	assert.Nil(t, sess2)
	assert.ErrorIs(t, err, errSessConflict)
}

func TestInMemorySessionManager_Retrieve(t *testing.T) {
	tests := []struct {
		name             string
		given            []DisplayScreenName
		lookupScreenName IdentScreenName
		wantScreenName   IdentScreenName
	}{
		{
			name: "lookup finds match",
			given: []DisplayScreenName{
				"user-screen-name-1",
				"user-screen-name-2",
			},
			lookupScreenName: NewIdentScreenName("user-screen-name-2"),
			wantScreenName:   NewIdentScreenName("user-screen-name-2"),
		},
		{
			name:             "lookup does not find match",
			given:            []DisplayScreenName{},
			lookupScreenName: NewIdentScreenName("user-screen-name-3"),
			wantScreenName:   NewIdentScreenName(""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewInMemorySessionManager(slog.Default())
			for _, screenName := range tt.given {
				sess, err := sm.AddSession(context.Background(), screenName)
				assert.NoError(t, err)
				sess.SetSignonComplete()
			}

			if have := sm.RetrieveSession(tt.lookupScreenName); have == nil {
				assert.Empty(t, tt.wantScreenName)
			} else {
				assert.Equal(t, tt.wantScreenName, have.IdentScreenName())
			}
		})
	}
}

func TestInMemorySessionManager_RetrieveSession_IncompleteSignon(t *testing.T) {
	sm := NewInMemorySessionManager(slog.Default())
	user1, err := sm.AddSession(context.Background(), "user-screen-name-1")
	assert.NoError(t, err)
	// user1 has not completed signon

	sess := sm.RetrieveSession(NewIdentScreenName("user-screen-name-1"))
	assert.Nil(t, sess, "should return nil for session with incomplete signon")

	user1.SetSignonComplete()
	sess = sm.RetrieveSession(NewIdentScreenName("user-screen-name-1"))
	assert.NotNil(t, sess, "should return session after signon is complete")
	assert.Equal(t, user1, sess)
}

func TestInMemorySessionManager_RetrieveSession_CompleteSignon(t *testing.T) {
	sm := NewInMemorySessionManager(slog.Default())
	user1, err := sm.AddSession(context.Background(), "user-screen-name-1")
	assert.NoError(t, err)
	user1.SetSignonComplete()

	sess := sm.RetrieveSession(NewIdentScreenName("user-screen-name-1"))
	assert.NotNil(t, sess)
	assert.Equal(t, user1, sess)
}

func TestInMemorySessionManager_RelayToScreenNames(t *testing.T) {
	sm := NewInMemorySessionManager(slog.Default())
	user1, err := sm.AddSession(context.Background(), "user-screen-name-1")
	assert.NoError(t, err)
	user1.SetSignonComplete()
	user2, err := sm.AddSession(context.Background(), "user-screen-name-2")
	assert.NoError(t, err)
	user2.SetSignonComplete()
	user3, err := sm.AddSession(context.Background(), "user-screen-name-3")
	assert.NoError(t, err)
	user3.SetSignonComplete()

	want := wire.SNACMessage{Frame: wire.SNACFrame{FoodGroup: wire.ICBM}}
	recips := []IdentScreenName{
		NewIdentScreenName("user-screen-name-1"),
		NewIdentScreenName("user-screen-name-2"),
	}
	sm.RelayToScreenNames(context.Background(), recips, want)

	have := <-user1.ReceiveMessage()
	assert.Equal(t, want, have)

	have = <-user2.ReceiveMessage()
	assert.Equal(t, want, have)

	<-user3.ReceiveMessage()
	assert.Fail(t, "user 3 should not receive a message")
}

func TestInMemorySessionManager_RelayToScreenNames_SkipIncompleteSignon(t *testing.T) {
	sm := NewInMemorySessionManager(slog.Default())
	user1, err := sm.AddSession(context.Background(), "user-screen-name-1")
	assert.NoError(t, err)
	user1.SetSignonComplete()

	user2, err := sm.AddSession(context.Background(), "user-screen-name-2")
	assert.NoError(t, err)
	// user2 has not completed signon

	user3, err := sm.AddSession(context.Background(), "user-screen-name-3")
	assert.NoError(t, err)
	user3.SetSignonComplete()

	want := wire.SNACMessage{Frame: wire.SNACFrame{FoodGroup: wire.ICBM}}
	recips := []IdentScreenName{
		NewIdentScreenName("user-screen-name-1"),
		NewIdentScreenName("user-screen-name-2"), // incomplete signon
		NewIdentScreenName("user-screen-name-3"),
	}
	sm.RelayToScreenNames(context.Background(), recips, want)

	have := <-user1.ReceiveMessage()
	assert.Equal(t, want, have)

	<-user2.ReceiveMessage()
	assert.Fail(t, "user 2 should not receive a message because signon is incomplete")

	have = <-user3.ReceiveMessage()
	assert.Equal(t, want, have)
}
