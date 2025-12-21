package state

import "time"

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
