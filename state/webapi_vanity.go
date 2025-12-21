package state

import (
	"database/sql"
	"log/slog"
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
