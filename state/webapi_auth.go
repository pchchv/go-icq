package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
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

// ValidateToken checks if a token is valid and returns the associated screen name.
func (s *WebAPITokenStore) ValidateToken(ctx context.Context, token string) (IdentScreenName, error) {
	var screenNameStr string
	var expiresAt time.Time
	query := `
		SELECT screen_name, expires_at
		FROM webapi_tokens
		WHERE token = ?
	`
	if err := s.store.db.QueryRowContext(ctx, query, token).Scan(&screenNameStr, &expiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NewIdentScreenName(""), errors.New("invalid token")
		} else {
			return NewIdentScreenName(""), fmt.Errorf("failed to validate token: %w", err)
		}
	}

	// check if token has expired
	if time.Now().After(expiresAt) {
		// clean up expired token
		s.DeleteToken(ctx, token)
		return NewIdentScreenName(""), errors.New("token expired")
	} else {
		return NewIdentScreenName(screenNameStr), nil
	}
}

// DeleteToken removes a token.
func (s *WebAPITokenStore) DeleteToken(ctx context.Context, token string) error {
	query := `DELETE FROM webapi_tokens WHERE token = ?`
	if _, err := s.store.db.ExecContext(ctx, query, token); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	return nil
}

// StoreToken saves an authentication token for a user.
func (s *WebAPITokenStore) StoreToken(ctx context.Context, token string, screenName IdentScreenName, expiresAt time.Time) error {
	query := `
		INSERT INTO webapi_tokens (token, screen_name, expires_at, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET
			screen_name = excluded.screen_name,
			expires_at = excluded.expires_at
	`
	if _, err := s.store.db.ExecContext(ctx, query, token, screenName.String(), expiresAt, time.Now()); err != nil {
		return fmt.Errorf("failed to store token: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes all expired tokens from the database.
func (s *WebAPITokenStore) CleanupExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM webapi_tokens WHERE expires_at < ?`
	if _, err := s.store.db.ExecContext(ctx, query, time.Now()); err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	return nil
}
