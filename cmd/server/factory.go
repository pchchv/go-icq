package main

import (
	"log/slog"

	"github.com/pchchv/go-icq/config"
	"github.com/pchchv/go-icq/foodgroup"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// Container groups together common dependencies.
type Container struct {
	cfg                    config.Config
	chatSessionManager     *state.InMemoryChatSessionManager
	hmacCookieBaker        state.HMACCookieBaker
	icbmSvc                *foodgroup.ICBMService
	inMemorySessionManager *state.InMemorySessionManager
	logger                 *slog.Logger
	rateLimitClasses       wire.RateLimitClasses
	snacRateLimits         wire.SNACRateLimits
	sqLiteUserStore        *state.SQLiteUserStore
	webAPISessionManager   *state.WebAPISessionManager
	Listeners              []config.Listener
}
