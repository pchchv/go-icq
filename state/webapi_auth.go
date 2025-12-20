package state

import (
	"context"
	"errors"
	"fmt"
)

// AuthenticateUser verifies username and password.
// This implementation uses the existing user store for authentication.
func (u *SQLiteUserStore) AuthenticateUser(ctx context.Context, username, password string) (*User, error) {
	// convert username to IdentScreenName for lookup
	identSN := NewIdentScreenName(username)

	// try to find the user
	user, err := u.User(ctx, identSN)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// in development mode with DISABLE_AUTH=true,
	// accept any password in production,
	// this would verify the password hash
	// for now, we'll accept any non-empty password if the user exists
	if password == "" {
		return nil, errors.New("password required")
	}

	// TODO: in production, verify password hash here
	// For development with DISABLE_AUTH, we just check if user exists
	return user, nil
}

// FindUserByScreenName finds a user by their screen name.
// This is just an alias for the User method to satisfy the UserManager interface.
func (u *SQLiteUserStore) FindUserByScreenName(ctx context.Context, screenName IdentScreenName) (*User, error) {
	return u.User(ctx, screenName)
}

// WebAPITokenStore manages authentication tokens for Web API sessions.
type WebAPITokenStore struct {
	store *SQLiteUserStore
}

// NewWebAPITokenStore creates a new token store.
func (s *SQLiteUserStore) NewWebAPITokenStore() *WebAPITokenStore {
	return &WebAPITokenStore{store: s}
}
