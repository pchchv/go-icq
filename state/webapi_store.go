package state

import (
	"context"
	"errors"
	"time"
)

// ErrDupAPIKey = errors.New("API key already exists")
var ErrNoAPIKey = errors.New("API key not found") // returned when an API key is not found

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
