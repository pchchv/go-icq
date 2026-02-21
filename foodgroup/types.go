package foodgroup

import (
	"context"
	"net/mail"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
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

// BARTItemManager is the interface for managing BART (Buddy Art) assets.
type BARTItemManager interface {
	// BARTItem retrieves a BART asset by its hash.
	BARTItem(ctx context.Context, hash []byte) ([]byte, error)
	// BuddyIconMetadata retrieves a user's buddy icon metadata. It returns nil
	// if the user does not have a buddy icon.
	BuddyIconMetadata(ctx context.Context, screenName state.IdentScreenName) (*wire.BARTID, error)
	// InsertBARTItem creates or updates a BART asset and blob hash.
	InsertBARTItem(ctx context.Context, hash []byte, blob []byte, itemType uint16) error
	// ListBARTItems returns BART assets filtered by type.
	ListBARTItems(ctx context.Context, itemType uint16) ([]state.BARTItem, error)
	// DeleteBARTItem deletes a BART asset by hash.
	DeleteBARTItem(ctx context.Context, hash []byte) error
}

// buddyBroadcaster defines methods for broadcasting buddy presence and visibility events
// to other sessions. These events notify users when a buddy comes online, goes offline,
// or changes visibility status.
type buddyBroadcaster interface {
	// BroadcastBuddyArrived notifies all relevant users that the given user has come online.
	BroadcastBuddyArrived(ctx context.Context, screenName state.IdentScreenName, userInfo wire.TLVUserInfo) error
	// BroadcastBuddyDeparted notifies all relevant users that the given user has gone offline.
	BroadcastBuddyDeparted(ctx context.Context, instance *state.SessionInstance) error
	// BroadcastVisibility sends presence updates to the specified filter list.
	// If sendDepartures is true, departure events are sent as well.
	BroadcastVisibility(ctx context.Context, you *state.SessionInstance, filter []state.IdentScreenName, sendDepartures bool) error
}
