package foodgroup

import (
	"context"
	"net/mail"

	"github.com/pchchv/go-icq/state"
)

// AccountManager is the interface for managing a user's account settings.
type AccountManager interface {
	// ConfirmStatus returns whether a user account has been confirmed.
	ConfirmStatus(ctx context.Context, screenName state.IdentScreenName) (bool, error)
	// EmailAddress looks up a user's email address by screen name.
	EmailAddress(ctx context.Context, screenName state.IdentScreenName) (*mail.Address, error)
	// RegStatus looks up a user's registration status by screen name.
	// It returns one of the following values:
	//   - wire.AdminInfoRegStatusFullDisclosure
	//   - wire.AdminInfoRegStatusLimitDisclosure
	//   - wire.AdminInfoRegStatusNoDisclosure
	RegStatus(ctx context.Context, screenName state.IdentScreenName) (uint16, error)
	// SetUserPassword sets the user's password hashes and auth key.
	SetUserPassword(ctx context.Context, screenName state.IdentScreenName, newPassword string) error
	// UpdateConfirmStatus sets whether a user account has been confirmed.
	UpdateConfirmStatus(ctx context.Context, screenName state.IdentScreenName, confirmStatus bool) error
	// UpdateDisplayScreenName updates the user's display screen name,
	// which is the screen name visible in the OSCAR client.
	// It derives the user identifier from the display screen name.
	UpdateDisplayScreenName(ctx context.Context, displayScreenName state.DisplayScreenName) error
	// UpdateEmailAddress changes a user's email address.
	UpdateEmailAddress(ctx context.Context, screenName state.IdentScreenName, emailAddress *mail.Address) error
	// UpdateRegStatus updates a user's registration status.
	// The regStatus param can be one of the following values:
	//   - wire.AdminInfoRegStatusFullDisclosure
	//   - wire.AdminInfoRegStatusLimitDisclosure
	//   - wire.AdminInfoRegStatusNoDisclosure
	UpdateRegStatus(ctx context.Context, screenName state.IdentScreenName, regStatus uint16) error
	// User returns all attributes for a user.
	User(ctx context.Context, screenName state.IdentScreenName) (*state.User, error)
}
