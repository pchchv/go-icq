package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// WebPreferenceManager handles Web API user preferences.
type WebPreferenceManager struct {
	store *SQLiteUserStore
}

// NewWebPreferenceManager creates a new WebPreferenceManager.
func (s *SQLiteUserStore) NewWebPreferenceManager() *WebPreferenceManager {
	return &WebPreferenceManager{store: s}
}

// GetPreferences retrieves user preferences from the database.
func (m *WebPreferenceManager) GetPreferences(ctx context.Context, screenName IdentScreenName) (map[string]interface{}, error) {
	var prefsJSON string
	q := `
		SELECT preferences
		FROM web_preferences
		WHERE screen_name = ?
	`
	if err := m.store.db.QueryRowContext(ctx, q, screenName.String()).Scan(&prefsJSON); err != nil {
		if err == sql.ErrNoRows {
			// return empty preferences if none exist
			return make(map[string]interface{}), nil
		} else {
			return nil, err
		}
	}

	var prefs map[string]interface{}
	if err := json.Unmarshal([]byte(prefsJSON), &prefs); err != nil {
		return nil, err
	}

	return prefs, nil
}

// SetPreferences stores user preferences in the database.
func (m *WebPreferenceManager) SetPreferences(ctx context.Context, screenName IdentScreenName, prefs map[string]interface{}) error {
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	q := `
		INSERT INTO web_preferences (screen_name, preferences, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (screen_name)
		DO UPDATE SET preferences = excluded.preferences, updated_at = excluded.updated_at
	`
	_, err = m.store.db.ExecContext(ctx, q, screenName.String(), string(prefsJSON), now, now)
	return err
}
