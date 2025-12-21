package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pchchv/go-icq/wire"
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

// WebPermitDenyManager handles Web API permit/deny list management.
type WebPermitDenyManager struct {
	store *SQLiteUserStore
}

// NewWebPermitDenyManager creates a new WebPermitDenyManager.
func (s *SQLiteUserStore) NewWebPermitDenyManager() *WebPermitDenyManager {
	return &WebPermitDenyManager{store: s}
}

// GetPDMode retrieves the permit/deny mode for a user.
func (m *WebPermitDenyManager) GetPDMode(ctx context.Context, screenName IdentScreenName) (wire.FeedbagPDMode, error) {
	var mode int
	q := `
		SELECT clientSidePDMode
		FROM buddyListMode
		WHERE screenName = ?
	`
	if err := m.store.db.QueryRowContext(ctx, q, screenName.String()).Scan(&mode); err != nil {
		if err == sql.ErrNoRows {
			// default to PermitAll if not set
			return wire.FeedbagPDModePermitAll, nil
		} else {
			return 0, err
		}
	}

	return wire.FeedbagPDMode(mode), nil
}

// GetPermitList retrieves the permit list for a user.
func (m *WebPermitDenyManager) GetPermitList(ctx context.Context, screenName IdentScreenName) ([]IdentScreenName, error) {
	q := `
		SELECT them
		FROM clientSideBuddyList
		WHERE me = ? AND isPermit = 1
	`
	rows, err := m.store.db.QueryContext(ctx, q, screenName.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []IdentScreenName
	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return nil, err
		}
		users = append(users, NewIdentScreenName(user))
	}

	return users, rows.Err()
}

// GetDenyList retrieves the deny list for a user.
func (m *WebPermitDenyManager) GetDenyList(ctx context.Context, screenName IdentScreenName) ([]IdentScreenName, error) {
	q := `
		SELECT them
		FROM clientSideBuddyList
		WHERE me = ? AND isDeny = 1
	`
	rows, err := m.store.db.QueryContext(ctx, q, screenName.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []IdentScreenName
	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return nil, err
		}
		users = append(users, NewIdentScreenName(user))
	}

	return users, rows.Err()
}

// SetPDMode sets the permit/deny mode for a user.
func (m *WebPermitDenyManager) SetPDMode(ctx context.Context, screenName IdentScreenName, mode wire.FeedbagPDMode) error {
	q := `
		INSERT INTO buddyListMode (screenName, clientSidePDMode)
		VALUES (?, ?)
		ON CONFLICT (screenName)
		DO UPDATE SET clientSidePDMode = excluded.clientSidePDMode
	`
	_, err := m.store.db.ExecContext(ctx, q, screenName.String(), int(mode))
	return err
}

// AddDenyBuddy adds a user to the deny list.
func (m *WebPermitDenyManager) AddDenyBuddy(ctx context.Context, me IdentScreenName, them IdentScreenName) error {
	q := `
		INSERT INTO clientSideBuddyList (me, them, isDeny)
		VALUES (?, ?, 1)
		ON CONFLICT (me, them) DO UPDATE SET isDeny = 1
	`
	_, err := m.store.db.ExecContext(ctx, q, me.String(), them.String())
	return err
}

// AddPermitBuddy adds a user to the permit list.
func (m *WebPermitDenyManager) AddPermitBuddy(ctx context.Context, me IdentScreenName, them IdentScreenName) error {
	q := `
		INSERT INTO clientSideBuddyList (me, them, isPermit)
		VALUES (?, ?, 1)
		ON CONFLICT (me, them) DO UPDATE SET isPermit = 1
	`
	_, err := m.store.db.ExecContext(ctx, q, me.String(), them.String())
	return err
}
