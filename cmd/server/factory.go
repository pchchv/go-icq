package main

import (
	"log/slog"
	"os"

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

// Helper function to check if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper function to get environment variable or return default.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
