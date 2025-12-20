package state

import (
	"database/sql"
	"log/slog"
	"sync"
	"time"
)

// APIUsageLog represents a single API request log entry.
type APIUsageLog struct {
	ID             int64     `json:"id"`
	DevID          string    `json:"dev_id"`
	Method         string    `json:"method"`
	Endpoint       string    `json:"endpoint"`
	Timestamp      time.Time `json:"timestamp"`
	ResponseTimeMs int       `json:"response_time_ms"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	ScreenName     string    `json:"screen_name,omitempty"`
	StatusCode     int       `json:"status_code"`
	IPAddress      string    `json:"ip_address"`
	UserAgent      string    `json:"user_agent"`
	RequestSize    int       `json:"request_size"`
	ResponseSize   int       `json:"response_size"`
}

// APIAnalytics provides analytics tracking for the Web API.
type APIAnalytics struct {
	db        *sql.DB
	logger    *slog.Logger
	buffer    []APIUsageLog
	bufferMu  sync.Mutex
	batchSize int
	ticker    *time.Ticker
	done      chan bool
}

// nullString returns a sql.NullString for the given string.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	} else {
		return sql.NullString{String: s, Valid: true}
	}
}
