package state

// WebPreferenceManager handles Web API user preferences.
type WebPreferenceManager struct {
	store *SQLiteUserStore
}

// NewWebPreferenceManager creates a new WebPreferenceManager.
func (s *SQLiteUserStore) NewWebPreferenceManager() *WebPreferenceManager {
	return &WebPreferenceManager{store: s}
}
