package state

import "strings"

// IdentScreenName struct stores the normalized version of a user's screen name.
// This format is used for uniformity in storage and comparison by removing spaces
// and converting all characters to lowercase.
type IdentScreenName struct {
	// screenName contains the identifier screen name value.
	// Do not assign this value directly.
	// Rather, set it through NewIdentScreenName.
	// This ensures that when an instance of IdentScreenName is present,
	// it's guaranteed to have a normalized value.
	screenName string
}

// NewIdentScreenName creates a new IdentScreenName.
func NewIdentScreenName(screenName string) IdentScreenName {
	str := strings.ReplaceAll(screenName, " ", "")
	str = strings.ToLower(str)
	return IdentScreenName{screenName: str}
}
