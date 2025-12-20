package state

import (
	"context"
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

// Close stops the analytics processor.
func (a *APIAnalytics) Close() {
	close(a.done)
	a.ticker.Stop()
}

// flush writes buffered logs to the database.
func (a *APIAnalytics) flush(ctx context.Context) {
	a.bufferMu.Lock()
	if len(a.buffer) == 0 {
		a.bufferMu.Unlock()
		return
	}

	// copy buffer and clear it
	logs := make([]APIUsageLog, len(a.buffer))
	copy(logs, a.buffer)
	a.buffer = a.buffer[:0]
	a.bufferMu.Unlock()

	// insert logs in a transaction
	tx, err := a.db.Begin()
	if err != nil {
		a.logger.Error("failed to begin transaction for analytics", "error", err)
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO api_usage_logs (
			dev_id, endpoint, method, timestamp, response_time_ms,
			status_code, ip_address, user_agent, screen_name,
			error_message, request_size, response_size
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		a.logger.Error("failed to prepare analytics insert statement", "error", err)
		return
	}
	defer stmt.Close()

	for _, log := range logs {
		_, err := stmt.Exec(
			log.DevID, log.Endpoint, log.Method, log.Timestamp.Unix(),
			log.ResponseTimeMs, log.StatusCode, log.IPAddress, log.UserAgent,
			nullString(log.ScreenName), nullString(log.ErrorMessage),
			log.RequestSize, log.ResponseSize,
		)
		if err != nil {
			a.logger.Error("failed to insert analytics log", "error", err)
		}
	}

	if err := tx.Commit(); err != nil {
		a.logger.Error("failed to commit analytics transaction", "error", err)
	}
}

// batchProcessor processes buffered logs in batches.
func (a *APIAnalytics) batchProcessor() {
	for {
		select {
		case <-a.ticker.C:
			a.flush(context.Background())
		case <-a.done:
			a.flush(context.Background()) // Final flush
			return
		}
	}
}

// nullString returns a sql.NullString for the given string.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	} else {
		return sql.NullString{String: s, Valid: true}
	}
}
