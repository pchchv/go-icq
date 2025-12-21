package state

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strings"
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

// APIQuota represents API usage quotas for a developer.
type APIQuota struct {
	DevID            string    `json:"dev_id"`
	DailyLimit       int       `json:"daily_limit"`
	MonthlyLimit     int       `json:"monthly_limit"`
	DailyUsed        int       `json:"daily_used"`
	MonthlyUsed      int       `json:"monthly_used"`
	LastResetDaily   time.Time `json:"last_reset_daily"`
	LastResetMonthly time.Time `json:"last_reset_monthly"`
	OverageAllowed   bool      `json:"overage_allowed"`
}

// APIUsageStats represents aggregated API usage statistics.
type APIUsageStats struct {
	DevID              string    `json:"dev_id"`
	Endpoint           string    `json:"endpoint"`
	PeriodType         string    `json:"period_type"`
	PeriodStart        time.Time `json:"period_start"`
	RequestCount       int       `json:"request_count"`
	ErrorCount         int       `json:"error_count"`
	TotalResponseTime  int       `json:"total_response_time_ms"`
	AvgResponseTime    int       `json:"avg_response_time_ms"`
	TotalRequestBytes  int64     `json:"total_request_bytes"`
	TotalResponseBytes int64     `json:"total_response_bytes"`
	UniqueUsers        int       `json:"unique_users"`
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

// NewAPIAnalytics creates a new API analytics instance.
func NewAPIAnalytics(db *sql.DB, logger *slog.Logger) *APIAnalytics {
	analytics := &APIAnalytics{
		db:        db,
		logger:    logger,
		batchSize: 100,
		buffer:    make([]APIUsageLog, 0, 100),
		ticker:    time.NewTicker(5 * time.Second),
		done:      make(chan bool),
	}

	// start background worker for batch processing
	go analytics.batchProcessor()

	return analytics
}

// Close stops the analytics processor.
func (a *APIAnalytics) Close() {
	close(a.done)
	a.ticker.Stop()
}

// LogRequest logs an API request asynchronously.
func (a *APIAnalytics) LogRequest(ctx context.Context, log APIUsageLog) {
	a.bufferMu.Lock()
	defer a.bufferMu.Unlock()

	a.buffer = append(a.buffer, log)
	// flush if buffer is full
	if len(a.buffer) >= a.batchSize {
		go a.flush(context.Background())
	}
}

// LogHTTPRequest logs an HTTP request with timing information.
func (a *APIAnalytics) LogHTTPRequest(
	ctx context.Context,
	r *http.Request,
	statusCode int,
	responseTime time.Duration,
	responseSize int,
	errorMsg string) {
	// extract IP address
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}

	// get request size
	var requestSize int
	if r.ContentLength > 0 {
		requestSize = int(r.ContentLength)
	}

	// extract dev_id from context (set by auth middleware)
	var devID string
	if val := r.Context().Value("dev_id"); val != nil {
		devID = val.(string)
	}

	// extract screen name if available
	var screenName string
	if val := r.Context().Value("screen_name"); val != nil {
		screenName = val.(string)
	}

	log := APIUsageLog{
		DevID:          devID,
		Endpoint:       r.URL.Path,
		Method:         r.Method,
		Timestamp:      time.Now(),
		ResponseTimeMs: int(responseTime.Milliseconds()),
		StatusCode:     statusCode,
		IPAddress:      ip,
		UserAgent:      r.UserAgent(),
		ScreenName:     screenName,
		ErrorMessage:   errorMsg,
		RequestSize:    requestSize,
		ResponseSize:   responseSize,
	}

	a.LogRequest(ctx, log)
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
			a.flush(context.Background())
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
