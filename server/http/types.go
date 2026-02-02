package http

import (
	"context"
	"net/mail"
	"time"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// BARTAssetManager defines methods for managing BART (Buddy ART) assets.
type BARTAssetManager interface {
	// BARTItem retrieves a BART asset by its hash.
	BARTItem(ctx context.Context, hash []byte) ([]byte, error)
	// InsertBARTItem inserts a BART asset.
	InsertBARTItem(ctx context.Context, hash []byte, blob []byte, itemType uint16) error
	// ListBARTItems returns BART assets filtered by type.
	ListBARTItems(ctx context.Context, itemType uint16) ([]state.BARTItem, error)
	// DeleteBARTItem deletes a BART asset by hash.
	DeleteBARTItem(ctx context.Context, hash []byte) error
}

// ChatSessionRetriever defines a method for
// retrieving all sessions associated with a specific chat room.
type ChatSessionRetriever interface {
	// AllSessions returns all active sessions in the chat room identified by cookie.
	AllSessions(cookie string) []*state.Session
}

// ChatRoomCreator defines a method for creating a new chat room.
type ChatRoomCreator interface {
	// CreateChatRoom creates a new chat room.
	CreateChatRoom(ctx context.Context, chatRoom *state.ChatRoom) error
}

// ChatRoomRetriever defines a method for retrieving all chat rooms under a specific exchange.
type ChatRoomRetriever interface {
	// AllChatRooms returns all chat rooms associated with the given exchange ID.
	AllChatRooms(ctx context.Context, exchange uint16) ([]state.ChatRoom, error)
}

// ChatRoomDeleter defines a method for deleting chat rooms.
type ChatRoomDeleter interface {
	// DeleteChatRooms deletes chat rooms by their names under a specific exchange.
	DeleteChatRooms(ctx context.Context, exchange uint16, names []string) error
}

// BuddyBroadcaster defines a method for broadcasting presence updates.
type BuddyBroadcaster interface {
	// BroadcastVisibility sends presence updates to the specified filter list.
	// If sendDepartures is true, departure events are sent as well.
	BroadcastVisibility(ctx context.Context, you *state.SessionInstance, filter []state.IdentScreenName, sendDepartures bool) error
}

// AccountManager defines methods for managing user account attributes such as email,
// confirmation status, registration status, and suspension.
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
	// UpdateSuspendedStatus updates the suspension status of a user account.
	UpdateSuspendedStatus(ctx context.Context, suspendedStatus uint16, screenName state.IdentScreenName) error
	// SetBotStatus updates the flag that indicates whether the user is a bot.
	SetBotStatus(ctx context.Context, isBot bool, screenName state.IdentScreenName) error
}

// FeedBagRetriever defines methods for retrieving buddy list metadata.
type FeedBagRetriever interface {
	// BuddyIconMetadata retrieves a user's buddy icon metadata.
	// It returns nil if the user does not have a buddy icon.
	BuddyIconMetadata(ctx context.Context, screenName state.IdentScreenName) (*wire.BARTID, error)
}

// FeedbagManager defines methods for managing feedbag (buddy list) entries.
// This interface matches foodgroup.FeedbagManager and is implemented by state.SQLiteUserStore.
type FeedbagManager interface {
	// Feedbag retrieves all feedbag items for a user.
	Feedbag(ctx context.Context, screenName state.IdentScreenName) ([]wire.FeedbagItem, error)
	// FeedbagUpsert inserts or updates feedbag items.
	FeedbagUpsert(ctx context.Context, screenName state.IdentScreenName, items []wire.FeedbagItem) error
	// FeedbagDelete deletes feedbag items.
	FeedbagDelete(ctx context.Context, screenName state.IdentScreenName, items []wire.FeedbagItem) error
}

// UserManager defines methods for accessing and inserting AIM user records.
type UserManager interface {
	// AllUsers returns all registered users.
	AllUsers(ctx context.Context) ([]state.User, error)
	// DeleteUser removes a user from the system by screen name.
	DeleteUser(ctx context.Context, screenName state.IdentScreenName) error
	// InsertUser inserts a new user into the system.
	// Return state.ErrDupUser if a user with the same screen name already exists.
	InsertUser(ctx context.Context, u state.User) error
	// SetUserPassword sets the user's password hashes and auth key.
	SetUserPassword(ctx context.Context, screenName state.IdentScreenName, newPassword string) error
	// User returns all attributes for a user.
	User(ctx context.Context, screenName state.IdentScreenName) (*state.User, error)
}

type aimChatUserHandle struct {
	ID         string `json:"id"`
	ScreenName string `json:"screen_name"`
}

type chatRoom struct {
	URL          string              `json:"url"`
	Name         string              `json:"name"`
	CreatorID    string              `json:"creator_id,omitempty"`
	CreateTime   time.Time           `json:"create_time"`
	Participants []aimChatUserHandle `json:"participants"`
}

type chatRoomCreate struct {
	Name string `json:"name"`
}

type chatRoomDelete struct {
	Names []string `json:"names"`
}

type userHandle struct {
	ID              string `json:"id"`
	IsICQ           bool   `json:"is_icq"`
	IsBot           bool   `json:"is_bot"`
	ScreenName      string `json:"screen_name"`
	SuspendedStatus string `json:"suspended_status"`
}

type userAccountHandle struct {
	ID              string `json:"id"`
	IsBot           bool   `json:"is_bot"`
	IsICQ           bool   `json:"is_icq"`
	Profile         string `json:"profile"`
	RegStatus       uint16 `json:"reg_status"`
	Confirmed       bool   `json:"confirmed"`
	ScreenName      string `json:"screen_name"`
	EmailAddress    string `json:"email_address"`
	SuspendedStatus string `json:"suspended_status"`
}

type userAccountPatch struct {
	IsBot               *bool   `json:"is_bot"`
	SuspendedStatusText *string `json:"suspended_status"`
}

type userWithPassword struct {
	ScreenName string `json:"screen_name"`
	Password   string `json:"password,omitempty"`
}
