package state

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
