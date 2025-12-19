package state

import (
	"context"
	"log/slog"
	"testing"

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
