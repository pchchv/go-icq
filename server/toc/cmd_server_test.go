package toc

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/mock"
)

func testOSCARProxy(t *testing.T) OSCARProxy {
	buddyService := newMockBuddyService(t)
	buddyService.EXPECT().
		BroadcastBuddyDeparted(mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	buddyListRegistry := newMockBuddyListRegistry(t)
	buddyListRegistry.EXPECT().
		UnregisterBuddyList(mock.Anything, mock.Anything).
		Maybe().
		Return(nil)
	authService := newMockAuthService(t)
	authService.EXPECT().
		Signout(mock.Anything, mock.Anything).
		Maybe()
	authService.EXPECT().
		SignoutChat(mock.Anything, mock.Anything).
		Maybe()
	return OSCARProxy{
		AuthService:       authService,
		BuddyListRegistry: buddyListRegistry,
		BuddyService:      buddyService,
		Logger:            slog.Default(),
	}
}
