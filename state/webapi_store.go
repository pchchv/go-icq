package state

import (
	"context"
	"errors"
	"time"
)

// ErrDupAPIKey = errors.New("API key already exists")
var ErrNoAPIKey = errors.New("API key not found") // returned when an API key is not found

// WebAPIKey represents a Web API authentication key.
type WebAPIKey struct {
	DevID          string     `json:"dev_id"`
	DevKey         string     `json:"dev_key"`
	AppName        string     `json:"app_name"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsed       *time.Time `json:"last_used,omitempty"`
	IsActive       bool       `json:"is_active"`
	RateLimit      int        `json:"rate_limit"`
	AllowedOrigins []string   `json:"allowed_origins"`
	Capabilities   []string   `json:"capabilities"`
}

// DeleteAPIKey removes an API key from the database.
func (f SQLiteUserStore) DeleteAPIKey(ctx context.Context, devID string) error {
	q := `
		DELETE FROM web_api_keys WHERE dev_id = ?
	`
	result, err := f.db.ExecContext(ctx, q, devID)
	if err != nil {
		return err
	}

	if rowsAffected, err := result.RowsAffected(); err != nil {
		return err
	} else if rowsAffected == 0 {
		return ErrNoAPIKey
	}

	return nil
}

// UpdateLastUsed updates the last_used timestamp for an API key.
func (f *SQLiteUserStore) UpdateLastUsed(ctx context.Context, devKey string) error {
	q := `
		UPDATE web_api_keys
		SET last_used = ?
		WHERE dev_key = ?
	`
	_, err := f.db.ExecContext(ctx, q, time.Now().Unix(), devKey)
	return err
}
