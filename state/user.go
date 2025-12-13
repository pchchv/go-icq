package state

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

var ErrICQUINInvalidFormat = errors.New("uin must be a number in the range 10000-2147483646")

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

// String returns the string representation of the IdentScreenName.
func (i IdentScreenName) String() string {
	return i.screenName
}

// UIN returns a numeric UIN representation of the IdentScreenName.
func (i IdentScreenName) UIN() uint32 {
	v, _ := strconv.Atoi(i.screenName)
	return uint32(v)
}

// DisplayScreenName type represents the screen name in the user-defined format.
// This includes the original casing and spacing as defined by the user.
type DisplayScreenName string

// IdentScreenName converts the DisplayScreenName to
// an IdentScreenName by applying the
// normalization process defined in NewIdentScreenName.
func (s DisplayScreenName) IdentScreenName() IdentScreenName {
	return NewIdentScreenName(string(s))
}

// String returns the original display string of the screen name,
// preserving the user-defined casing and spaces.
func (s DisplayScreenName) String() string {
	return string(s)
}

// IsUIN indicates whether the screen name is an ICQ UIN.
func (s DisplayScreenName) IsUIN() bool {
	if len(s) == 0 {
		return false
	}

	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

// ValidateUIN returns an error if the instance is not a valid ICQ UIN.
// Possible error is an ErrICQUINInvalidFormat
// (if the UIN is not a number or is not in the valid range).
func (s DisplayScreenName) ValidateUIN() error {
	uin, err := strconv.Atoi(string(s))
	if err != nil || uin < 10000 || uin > 2147483646 {
		return ErrICQUINInvalidFormat
	}
	return nil
}
