package state

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

var (
	ErrAIMHandleLength        = errors.New("screen name must be between 3 and 16 characters")
	ErrICQUINInvalidFormat    = errors.New("uin must be a number in the range 10000-2147483646")
	ErrAIMHandleInvalidFormat = errors.New("screen name must start with a letter, cannot end with a space, and must contain only letters, numbers, and spaces")
)

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

// ValidateAIMHandle returns an error if the instance is not a valid AIM screen name.
// Possible errors:
//   - ErrAIMHandleLength: if the screen name has less than 3 non-space
//     characters or more than 16 characters (including spaces).
//   - ErrAIMHandleInvalidFormat: if the screen name does not start with a letter,
//     ends with a space, or contains invalid characters.
func (s DisplayScreenName) ValidateAIMHandle() error {
	if len(s) > 16 {
		return ErrAIMHandleLength
	}

	var c int
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			c++
		} else if r != ' ' {
			return ErrAIMHandleInvalidFormat
		}
	}

	if c < 3 {
		return ErrAIMHandleLength
	}

	// Must start with a letter, cannot end with a space,
	// and must contain only letters, numbers, and spaces.
	if !unicode.IsLetter(rune(s[0])) || s[len(s)-1] == ' ' {
		return ErrAIMHandleInvalidFormat
	}

	return nil
}
