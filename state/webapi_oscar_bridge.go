package state

// OSCARBridgeStore manages the persistence of OSCAR bridge sessions in the database.
// It provides methods to store, retrieve,
// and manage the mapping between WebAPI sessions and OSCAR authentication cookies.
type OSCARBridgeStore struct {
	store *SQLiteUserStore
}

// NewOSCARBridgeStore creates a new OSCAR bridge store instance.
func (s *SQLiteUserStore) NewOSCARBridgeStore() *OSCARBridgeStore {
	return &OSCARBridgeStore{store: s}
}
