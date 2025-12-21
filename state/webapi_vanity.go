package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// VanityInfo represents the response for vanity URL lookups.
type VanityInfo struct {
	Bio         string                 `json:"bio,omitempty"`
	Website     string                 `json:"website,omitempty"`
	Location    string                 `json:"location,omitempty"`
	VanityURL   string                 `json:"vanityUrl"`
	ScreenName  string                 `json:"screenName"`
	DisplayName string                 `json:"displayName,omitempty"`
	ProfileURL  string                 `json:"profileUrl"`
	IsActive    bool                   `json:"isActive"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// VanityURL represents a user's vanity URL configuration.
type VanityURL struct {
	Bio          string     `json:"bio,omitempty"`
	Website      string     `json:"website,omitempty"`
	Location     string     `json:"location,omitempty"`
	IsActive     bool       `json:"isActive"`
	VanityURL    string     `json:"vanityUrl"`
	ClickCount   int        `json:"clickCount"`
	ScreenName   string     `json:"screenName"`
	DisplayName  string     `json:"displayName,omitempty"`
	LastAccessed *time.Time `json:"lastAccessed,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// VanityURLRedirect represents a vanity URL access record.
type VanityURLRedirect struct {
	ID         int64     `json:"id"`
	VanityURL  string    `json:"vanityUrl"`
	AccessedAt time.Time `json:"accessedAt"`
	IPAddress  string    `json:"ipAddress,omitempty"`
	UserAgent  string    `json:"userAgent,omitempty"`
	Referer    string    `json:"referer,omitempty"`
}

// VanityURLManager manages vanity URL operations.
type VanityURLManager struct {
	db       *sql.DB
	logger   *slog.Logger
	baseURL  string   // Base URL for the service (e.g., "https://aim.example.com")
	reserved []string // Reserved URLs that cannot be claimed
}

// NewVanityURLManager creates a new vanity URL manager.
func NewVanityURLManager(db *sql.DB, logger *slog.Logger, baseURL string) *VanityURLManager {
	return &VanityURLManager{
		db:      db,
		logger:  logger,
		baseURL: baseURL,
		reserved: []string{
			"api", "admin", "help", "support", "about", "terms", "privacy",
			"login", "logout", "register", "signup", "signin", "settings",
			"profile", "user", "users", "aim", "aol", "webapi", "oscar",
			"chat", "im", "message", "buddy", "buddies", "feed", "rss",
		},
	}
}

// CreateOrUpdateVanityURL creates or updates a vanity URL for a user.
func (m *VanityURLManager) CreateOrUpdateVanityURL(ctx context.Context, screenName string, vanityURL string, info map[string]interface{}) error {
	// calidate vanity URL
	if err := m.validateVanityURL(vanityURL); err != nil {
		return err
	}

	// check if URL is reserved
	if m.isReserved(vanityURL) {
		return fmt.Errorf("vanity URL '%s' is reserved", vanityURL)
	}

	// extract optional fields from info
	displayName, _ := info["displayName"].(string)
	bio, _ := info["bio"].(string)
	location, _ := info["location"].(string)
	website, _ := info["website"].(string)
	now := time.Now()
	// try to update existing record first
	updateQuery := `
		UPDATE vanity_urls
		SET vanity_url = ?, display_name = ?, bio = ?, location = ?,
		    website = ?, updated_at = ?, is_active = ?
		WHERE screen_name = ?
	`
	result, err := m.db.ExecContext(ctx, updateQuery,
		vanityURL, displayName, bio, location, website,
		now.Unix(), true, screenName,
	)
	if err != nil {
		return fmt.Errorf("failed to update vanity URL: %w", err)
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
		m.logger.InfoContext(ctx, "updated vanity URL",
			"screenName", screenName,
			"vanityURL", vanityURL,
		)
		return nil
	}

	// insert new record
	insertQuery := `
		INSERT INTO vanity_urls (
			screen_name, vanity_url, display_name, bio, location,
			website, created_at, updated_at, is_active, click_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = m.db.ExecContext(ctx, insertQuery,
		screenName, vanityURL, displayName, bio, location,
		website, now.Unix(), now.Unix(), true, 0,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("vanity URL '%s' is already taken", vanityURL)
		}
		return fmt.Errorf("failed to create vanity URL: %w", err)
	}

	m.logger.InfoContext(ctx, "created vanity URL",
		"screenName", screenName,
		"vanityURL", vanityURL,
	)

	return nil
}

// GetVanityInfo retrieves vanity URL information.
func (m *VanityURLManager) GetVanityInfo(ctx context.Context, vanityURL string) (*VanityInfo, error) {
	var v VanityURL
	var createdAt, updatedAt int64
	var lastAccessed sql.NullInt64
	// clean the vanity URL
	vanityURL = strings.ToLower(strings.TrimSpace(vanityURL))
	query := `
		SELECT screen_name, vanity_url, display_name, bio, location,
		       website, created_at, updated_at, is_active, click_count, last_accessed
		FROM vanity_urls
		WHERE vanity_url = ? AND is_active = 1
	`
	err := m.db.QueryRowContext(ctx, query, vanityURL).Scan(
		&v.ScreenName, &v.VanityURL, &v.DisplayName, &v.Bio, &v.Location,
		&v.Website, &createdAt, &updatedAt, &v.IsActive, &v.ClickCount, &lastAccessed,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("vanity URL not found: %s", vanityURL)
		}
		return nil, fmt.Errorf("failed to get vanity info: %w", err)
	}

	v.CreatedAt = time.Unix(createdAt, 0)
	v.UpdatedAt = time.Unix(updatedAt, 0)
	if lastAccessed.Valid {
		t := time.Unix(lastAccessed.Int64, 0)
		v.LastAccessed = &t
	}

	// create response info
	info := &VanityInfo{
		ScreenName:  v.ScreenName,
		VanityURL:   v.VanityURL,
		DisplayName: v.DisplayName,
		Bio:         v.Bio,
		Location:    v.Location,
		Website:     v.Website,
		ProfileURL:  m.buildProfileURL(v.VanityURL),
		IsActive:    v.IsActive,
		Extra: map[string]interface{}{
			"createdAt":  v.CreatedAt.Unix(),
			"clickCount": v.ClickCount,
		},
	}

	// update click count and last accessed asynchronously
	go m.recordAccess(context.Background(), vanityURL)

	return info, nil
}

// isReserved checks if a vanity URL is in the reserved list.
func (m *VanityURLManager) isReserved(vanityURL string) bool {
	vanityURL = strings.ToLower(vanityURL)
	for _, reserved := range m.reserved {
		if reserved == vanityURL {
			return true
		}
	}
	return false
}

// validateVanityURL validates the format of a vanity URL.
func (m *VanityURLManager) validateVanityURL(vanityURL string) error {
	// clean and lowercase
	vanityURL = strings.ToLower(strings.TrimSpace(vanityURL))
	// check length
	if len(vanityURL) < 3 || len(vanityURL) > 30 {
		return errors.New("vanity URL must be between 3 and 30 characters")
	}

	// check format (alphanumeric, hyphens, underscores only)
	if validFormat := regexp.MustCompile(`^[a-z0-9_-]+$`); !validFormat.MatchString(vanityURL) {
		return errors.New("vanity URL can only contain letters, numbers, hyphens, and underscores")
	}

	// can't start or end with special characters
	if strings.HasPrefix(vanityURL, "-") || strings.HasPrefix(vanityURL, "_") || strings.HasSuffix(vanityURL, "-") || strings.HasSuffix(vanityURL, "_") {
		return errors.New("vanity URL cannot start or end with hyphens or underscores")
	}

	return nil
}

// buildProfileURL builds the full profile URL for a vanity URL.
func (m *VanityURLManager) buildProfileURL(vanityURL string) string {
	if m.baseURL == "" {
		return fmt.Sprintf("/profile/%s", vanityURL)
	} else {
		return fmt.Sprintf("%s/profile/%s", strings.TrimRight(m.baseURL, "/"), vanityURL)
	}
}

// recordAccess records a vanity URL access.
func (m *VanityURLManager) recordAccess(ctx context.Context, vanityURL string) {
	// update click count and last accessed time
	updateQuery := `
		UPDATE vanity_urls
		SET click_count = click_count + 1, last_accessed = ?
		WHERE vanity_url = ?
	`
	if _, err := m.db.ExecContext(ctx, updateQuery, time.Now().Unix(), vanityURL); err != nil {
		m.logger.Error("failed to record vanity URL access", "error", err, "vanityURL", vanityURL)
	}
}
