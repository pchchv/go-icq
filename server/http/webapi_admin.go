package http

import (
	"context"

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
