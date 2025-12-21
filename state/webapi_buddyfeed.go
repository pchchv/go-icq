package state

import (
	"database/sql"
	"log/slog"
	"time"
)

// BuddyFeed represents a user's feed configuration.
type BuddyFeed struct {
	ID          int64     `json:"id"`
	ScreenName  string    `json:"screenName"`
	FeedType    string    `json:"feedType"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	PublishedAt time.Time `json:"publishedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	IsActive    bool      `json:"isActive"`
}

// BuddyFeedItem represents an individual feed entry.
type BuddyFeedItem struct {
	ID          int64     `json:"id"`
	FeedID      int64     `json:"feedId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	GUID        string    `json:"guid"`
	Author      string    `json:"author"`
	Categories  []string  `json:"categories"`
	PublishedAt time.Time `json:"publishedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// BuddyFeedSubscription represents a feed subscription.
type BuddyFeedSubscription struct {
	ID                   int64      `json:"id"`
	FeedID               int64      `json:"feedId"`
	SubscribedAt         time.Time  `json:"subscribedAt"`
	LastCheckedAt        *time.Time `json:"lastCheckedAt"`
	SubscriberScreenName string     `json:"subscriberScreenName"`
}

// BuddyFeedManager manages buddy feed operations.
type BuddyFeedManager struct {
	db     *sql.DB
	logger *slog.Logger
}

// NewBuddyFeedManager creates a new buddy feed manager.
func NewBuddyFeedManager(db *sql.DB, logger *slog.Logger) *BuddyFeedManager {
	return &BuddyFeedManager{
		db:     db,
		logger: logger,
	}
}
