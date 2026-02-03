package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/state"
)

// WebAPIKeyManager defines methods for managing Web API authentication keys.
type WebAPIKeyManager interface {
	// CreateAPIKey creates a new Web API key.
	CreateAPIKey(ctx context.Context, key state.WebAPIKey) error
	// GetAPIKeyByDevID retrieves an API key by its developer ID.
	GetAPIKeyByDevID(ctx context.Context, devID string) (*state.WebAPIKey, error)
	// ListAPIKeys returns all Web API keys.
	ListAPIKeys(ctx context.Context) ([]state.WebAPIKey, error)
	// UpdateAPIKey updates an existing Web API key.
	UpdateAPIKey(ctx context.Context, devID string, updates state.WebAPIKeyUpdate) error
	// DeleteAPIKey removes a Web API key.
	DeleteAPIKey(ctx context.Context, devID string) error
}

// postWebAPIKeyHandler handles POST /admin/webapi/keys requests.
func postWebAPIKeyHandler(w http.ResponseWriter, r *http.Request, keyManager WebAPIKeyManager, newUUID func() uuid.UUID, logger *slog.Logger) {
	var req createWebAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "malformed request body", http.StatusBadRequest)
		return
	}

	// validate required fields
	if req.AppName == "" {
		http.Error(w, "app_name is required", http.StatusBadRequest)
		return
	}

	// set defaults
	if req.RateLimit <= 0 {
		req.RateLimit = 60 // default rate limit
	}

	// generate secure API key
	keyBytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(keyBytes); err != nil {
		logger.Error("failed to generate API key", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	devKey := hex.EncodeToString(keyBytes)

	// generate developer ID
	devID := fmt.Sprintf("dev_%s", newUUID().String())
	// create the API key record
	apiKey := state.WebAPIKey{
		DevID:          devID,
		DevKey:         devKey,
		AppName:        req.AppName,
		CreatedAt:      time.Now(),
		IsActive:       true,
		RateLimit:      req.RateLimit,
		AllowedOrigins: req.AllowedOrigins,
		Capabilities:   req.Capabilities,
	}
	// save to database
	if err := keyManager.CreateAPIKey(r.Context(), apiKey); err != nil {
		if err == state.ErrDupAPIKey {
			http.Error(w, "API key already exists", http.StatusConflict)
			return
		}

		logger.Error("failed to create API key", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// return the created key (including the dev_key which is only shown once)
	resp := webAPIKeyResponse{
		DevID:          apiKey.DevID,
		DevKey:         apiKey.DevKey, // only shown on creation
		AppName:        apiKey.AppName,
		CreatedAt:      apiKey.CreatedAt,
		IsActive:       apiKey.IsActive,
		RateLimit:      apiKey.RateLimit,
		Capabilities:   apiKey.Capabilities,
		AllowedOrigins: apiKey.AllowedOrigins,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "err", err.Error())
	}
}

// getWebAPIKeyHandler handles GET /admin/webapi/keys/{id} requests.
func getWebAPIKeyHandler(w http.ResponseWriter, r *http.Request, keyManager WebAPIKeyManager, logger *slog.Logger) {
	devID := r.PathValue("id")
	if devID == "" {
		http.Error(w, "missing developer ID", http.StatusBadRequest)
		return
	}

	key, err := keyManager.GetAPIKeyByDevID(r.Context(), devID)
	if err != nil {
		if err == state.ErrNoAPIKey {
			http.Error(w, "API key not found", http.StatusNotFound)
			return
		}

		logger.Error("failed to get API key", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// convert to response format (without dev_key)
	resp := webAPIKeyResponse{
		DevID:          key.DevID,
		AppName:        key.AppName,
		CreatedAt:      key.CreatedAt,
		LastUsed:       key.LastUsed,
		IsActive:       key.IsActive,
		RateLimit:      key.RateLimit,
		AllowedOrigins: key.AllowedOrigins,
		Capabilities:   key.Capabilities,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "err", err.Error())
	}
}

// getWebAPIKeysHandler handles GET /admin/webapi/keys requests.
func getWebAPIKeysHandler(w http.ResponseWriter, r *http.Request, keyManager WebAPIKeyManager, logger *slog.Logger) {
	keys, err := keyManager.ListAPIKeys(r.Context())
	if err != nil {
		logger.Error("failed to list API keys", "err", err.Error())
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// convert to response format (without dev_key)
	resp := make([]webAPIKeyResponse, 0, len(keys))
	for _, key := range keys {
		resp = append(resp, webAPIKeyResponse{
			DevID:          key.DevID,
			AppName:        key.AppName,
			CreatedAt:      key.CreatedAt,
			LastUsed:       key.LastUsed,
			IsActive:       key.IsActive,
			RateLimit:      key.RateLimit,
			AllowedOrigins: key.AllowedOrigins,
			Capabilities:   key.Capabilities,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Error("failed to encode response", "err", err.Error())
	}
}
