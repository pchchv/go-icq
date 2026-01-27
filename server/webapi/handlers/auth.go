package handlers

import (
	"context"
	"time"

	"github.com/pchchv/go-icq/state"
)

// TokenStore manages authentication tokens.
type TokenStore interface {
	// StoreToken saves an authentication token for a user
	StoreToken(ctx context.Context, token string, screenName state.IdentScreenName, expiresAt time.Time) error
	// ValidateToken checks if a token is valid and returns the associated screen name
	ValidateToken(ctx context.Context, token string) (state.IdentScreenName, error)
	// DeleteToken removes a token
	DeleteToken(ctx context.Context, token string) error
}

// UserManager defines methods for user authentication.
type UserManager interface {
	// AuthenticateUser verifies username and password.
	AuthenticateUser(ctx context.Context, username, password string) (*state.User, error)
	// FindUserByScreenName finds a user by their screen name.
	FindUserByScreenName(ctx context.Context, screenName state.IdentScreenName) (*state.User, error)
	// InsertUser creates a new user (for DISABLE_AUTH mode).
	InsertUser(ctx context.Context, u state.User) error
}
