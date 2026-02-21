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

// ChatMessageRelayer defines the interface for sending messages to chat room participants.
type ChatMessageRelayer interface {
	// AllSessions returns all chat room participants.
	// Returns ErrChatRoomNotFound if the room does not exist.
	AllSessions(chatCookie string) []*state.Session
	// RelayToAllExcept sends a message to all chat room participants except for the
	// participant with a particular screen name.
	// Returns ErrChatRoomNotFound if the room does not exist for cookie.
	RelayToAllExcept(ctx context.Context, chatCookie string, except state.IdentScreenName, msg wire.SNACMessage)
	// RelayToScreenName sends a message to a chat room user.
	// Returns ErrChatRoomNotFound if the room does not exist for cookie.
	RelayToScreenName(ctx context.Context, chatCookie string, recipient state.IdentScreenName, msg wire.SNACMessage)
}

// ChatRoomRegistry defines the interface for storing and retrieving chat
// rooms in a persistent store. The persistent store has two purposes:
//   - Remember user-created chat rooms (exchange 4) so that clients can reconnect to the rooms following server restarts.
//   - Keep track of public chat room created by the server operator (exchange 5).
//     User's can only join public chat rooms that exist in the room registry.
type ChatRoomRegistry interface {
	// ChatRoomByCookie looks up a chat room by exchange.
	// Returns state.ErrChatRoomNotFound if the room does not exist for cookie.
	ChatRoomByCookie(ctx context.Context, chatCookie string) (state.ChatRoom, error)
	// ChatRoomByName looks up a chat room by exchange and name.
	// Returns state.ErrChatRoomNotFound if the room does not exist for exchange and name.
	ChatRoomByName(ctx context.Context, exchange uint16, name string) (state.ChatRoom, error)
	// CreateChatRoom creates a new chat room.
	CreateChatRoom(ctx context.Context, chatRoom *state.ChatRoom) error
}

// ChatSessionRegistry defines the interface for adding and removing chat sessions.
type ChatSessionRegistry interface {
	// AddSession adds a session to the chat session manager.
	// The chatCookie param identifies the chat room to which screenName is added.
	// It returns the newly created session instance registered in the chat session manager.
	AddSession(ctx context.Context, chatCookie string, screenName state.DisplayScreenName) (*state.SessionInstance, error)
	// RemoveSession removes a session from the chat session manager.
	RemoveSession(instance *state.SessionInstance)
}

// ClientSideBuddyListManager defines operations for managing a user's buddy list,
// including permissions like allow/deny and buddy group modifications.
//
// This interface manages a client-side buddy list that is
// temporarily copied to the server on a per-session basis.
// The list is cleared when the user signs out.
type ClientSideBuddyListManager interface {
	// AddBuddy adds 'them' to 'me's buddy list.
	AddBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	// DenyBuddy blocks messages and presence visibility from 'them' to 'me'.
	DenyBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	// PermitBuddy allows messages and presence visibility from 'them' to 'me',
	// potentially overriding deny settings.
	PermitBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	// RemoveBuddy removes 'them' from 'me's buddy list.
	// Does not affect permit/deny status.
	RemoveBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	// RemoveDenyBuddy removes 'them' from 'me's deny list,
	// restoring default visibility and messaging behavior.
	RemoveDenyBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	// RemovePermitBuddy removes 'them' from 'me's permit list,
	// restoring default visibility and messaging behavior.
	RemovePermitBuddy(ctx context.Context, me state.IdentScreenName, them state.IdentScreenName) error
	// SetPDMode sets the permit/deny mode (e.g., allow all, deny all, permit some) for 'me'.
	// It clears any existing permit/deny records.
	SetPDMode(ctx context.Context, me state.IdentScreenName, pdMode wire.FeedbagPDMode) error
}

// CookieBaker defines methods for issuing and verifying AIM authentication tokens ("cookies").
// These tokens are used for authenticating client sessions with AIM services.
type CookieBaker interface {
	// Crack verifies and decodes a previously issued authentication token.
	// Returns the original payload if the token is valid.
	Crack(data []byte) ([]byte, error)
	// Issue creates a new authentication token from the given payload.
	// The resulting token can later be verified using Crack.
	Issue(data []byte) ([]byte, error)
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
