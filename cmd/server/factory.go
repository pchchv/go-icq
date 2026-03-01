package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

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

func validateConfigMigration() error {
	var newEnvVarsMissing, oldEnvVarsFound []string
	// old environment variables that should be removed
	oldEnvVars := []string{
		"API_HOST",
		"API_PORT",
		"KERBEROS_PORT",
		"ALERT_PORT",
		"AUTH_PORT",
		"BART_PORT",
		"BOS_PORT",
		"CHAT_NAV_PORT",
		"CHAT_PORT",
		"ADMIN_PORT",
		"ODIR_PORT",
		"OSCAR_HOST",
		"TOC_HOST",
		"TOC_PORT",
	}
	// new environment variables that should be present
	newEnvVars := []string{
		"API_LISTENER",
		"OSCAR_ADVERTISED_LISTENERS_PLAIN",
		"OSCAR_LISTENERS",
		"TOC_LISTENERS",
	}
	// check for old environment variables that should be removed
	for _, envVar := range oldEnvVars {
		if os.Getenv(envVar) != "" {
			oldEnvVarsFound = append(oldEnvVarsFound, envVar)
		}
	}

	// check for new environment variables that should be present
	for _, envVar := range newEnvVars {
		if os.Getenv(envVar) == "" {
			newEnvVarsMissing = append(newEnvVarsMissing, envVar)
		}
	}

	// if there are any issues, return an error with details
	if len(oldEnvVarsFound) > 0 || len(newEnvVarsMissing) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("Open OSCAR Server v0.19.0 introduced some breaking configuration changes that you need to fix.\n")
		if len(oldEnvVarsFound) > 0 {
			errorMsg.WriteString("\nOld environment variables that must be removed:\n\n")
			for _, envVar := range oldEnvVarsFound {
				errorMsg.WriteString(fmt.Sprintf("  - %s\n", envVar))
			}
		}

		if len(newEnvVarsMissing) > 0 {
			errorMsg.WriteString("\nNew environment variables that must be provided:\n\n")
			for _, envVar := range newEnvVarsMissing {
				errorMsg.WriteString(fmt.Sprintf("  - %s\n", envVar))
			}

			// generate export commands based on old environment variables
			errorMsg.WriteString("\nCopy/paste this updated configuration into your settings file:\n\n")
			if contains(newEnvVarsMissing, "API_LISTENER") {
				apiHost := getEnvOrDefault("API_HOST", "127.0.0.1")
				apiPort := getEnvOrDefault("API_PORT", "8080")
				errorMsg.WriteString(fmt.Sprintf("export API_LISTENER=%s:%s\n", apiHost, apiPort))
			}

			if contains(newEnvVarsMissing, "OSCAR_ADVERTISED_LISTENERS_PLAIN") {
				oscarHost := getEnvOrDefault("OSCAR_HOST", "127.0.0.1")
				authPort := getEnvOrDefault("AUTH_PORT", "5190")
				errorMsg.WriteString(fmt.Sprintf("export OSCAR_ADVERTISED_LISTENERS_PLAIN=LOCAL://%s:%s\n", oscarHost, authPort))
			}

			if contains(newEnvVarsMissing, "OSCAR_LISTENERS") {
				authPort := getEnvOrDefault("AUTH_PORT", "5190")
				errorMsg.WriteString(fmt.Sprintf("export OSCAR_LISTENERS=LOCAL://0.0.0.0:%s\n", authPort))
			}

			if contains(newEnvVarsMissing, "KERBEROS_LISTENERS") {
				kerberosPort := getEnvOrDefault("KERBEROS_PORT", "1088")
				errorMsg.WriteString(fmt.Sprintf("export KERBEROS_LISTENERS=LOCAL://0.0.0.0:%s\n", kerberosPort))
			}

			if contains(newEnvVarsMissing, "TOC_LISTENERS") {
				tocHost := getEnvOrDefault("TOC_HOST", "0.0.0.0")
				tocPort := getEnvOrDefault("TOC_PORT", "9898")
				errorMsg.WriteString(fmt.Sprintf("export TOC_LISTENERS=%s:%s\n", tocHost, tocPort))
			}
		}

		return errors.New(errorMsg.String())
	}

	return nil
}
