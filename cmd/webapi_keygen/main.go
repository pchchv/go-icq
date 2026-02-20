package main

import (
	"os"
	"strings"

	"github.com/pchchv/go-icq/state"
)

func parseCSV(input string) []string {
	if input == "" {
		return []string{}
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func connectToStore() (*state.SQLiteUserStore, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "oscar.sqlite"
	}

	return state.NewSQLiteUserStore(dbPath)
}
