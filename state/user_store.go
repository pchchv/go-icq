package state

import "database/sql"

// SQLiteUserStore stores user feedbag (buddy list), profile,
// and authentication credentials information in a SQLite database.
type SQLiteUserStore struct {
	db *sql.DB
}
