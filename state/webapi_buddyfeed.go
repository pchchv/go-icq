package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// CreateFeed creates a new buddy feed.
func (m *BuddyFeedManager) CreateFeed(ctx context.Context, feed BuddyFeed) (*BuddyFeed, error) {
	var id int64
	now := time.Now()
	query := `
		INSERT INTO buddy_feeds (
			screen_name, feed_type, title, description, link,
			published_at, created_at, updated_at, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`
	err := m.db.QueryRowContext(ctx, query,
		feed.ScreenName, feed.FeedType, feed.Title, feed.Description, feed.Link,
		feed.PublishedAt.Unix(), now.Unix(), now.Unix(), feed.IsActive,
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create feed: %w", err)
	}

	feed.ID = id
	feed.CreatedAt = now
	feed.UpdatedAt = now

	return &feed, nil
}

// AddFeedItem adds a new item to a feed.
func (m *BuddyFeedManager) AddFeedItem(ctx context.Context, feedID int64, item BuddyFeedItem) (*BuddyFeedItem, error) {
	var id int64
	now := time.Now()
	categoriesJSON, _ := json.Marshal(item.Categories)
	query := `
		INSERT INTO buddy_feed_items (
			feed_id, title, description, link, guid,
			author, categories, published_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`
	err := m.db.QueryRowContext(ctx, query,
		feedID, item.Title, item.Description, item.Link, item.GUID,
		item.Author, string(categoriesJSON), item.PublishedAt.Unix(), now.Unix(),
	).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to add feed item: %w", err)
	}

	item.ID = id
	item.FeedID = feedID
	item.CreatedAt = now

	// update feed's updated_at timestamp
	updateQuery := `UPDATE buddy_feeds SET updated_at = ? WHERE id = ?`
	m.db.ExecContext(ctx, updateQuery, now.Unix(), feedID)

	return &item, nil
}
