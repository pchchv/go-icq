package http

import (
	"net/mail"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// RegStatusParams is the list of parameters passed at
// the mock accountManager.RegStatus call site.
type RegStatusParams []struct {
	screenName state.IdentScreenName
	result     uint16
	err        error
}

// ConfirmStatusParams is the list of parameters passed at
// the mock accountManager.ConfirmStatus call site.
type ConfirmStatusParams []struct {
	screenName state.IdentScreenName
	result     bool
	err        error
}

// EmailAddressParams is the list of parameters passed at
// the mock accountManager.EmailAddress call site.
type EmailAddressParams []struct {
	screenName state.IdentScreenName
	result     *mail.Address
	err        error
}

// updateSuspendedStatus is the list of parameters passed at
// the mock accountManager.updateSuspendedStatus call site.
type updateSuspendedStatusParams []struct {
	suspendedStatus uint16
	screenName      state.IdentScreenName
	err             error
}

// setBotStatusParams is the list of parameters passed at
// the mock accountManager.SetBotStatus call site.
type setBotStatusParams []struct {
	isBot      bool
	screenName state.IdentScreenName
	err        error
}

// accountManagerParams is a helper struct that
// contains mock parameters for accountManager methods.
type accountManagerParams struct {
	RegStatusParams
	EmailAddressParams
	ConfirmStatusParams
	setBotStatusParams
	updateSuspendedStatusParams
}

// allChatRoomsParams is the list of parameters passed at
// the mock ChatRoomRetriever.AllChatRooms call site.
type allChatRoomsParams []struct {
	exchange uint16
	result   []state.ChatRoom
	err      error
}

// chatRoomRetrieverParams is a helper struct that
// contains mock parameters for ChatRoomRetriever methods.
type chatRoomRetrieverParams struct {
	allChatRoomsParams
}

// deleteChatRoomsParams is the list of parameters passed at
// the mock ChatRoomDeleter.DeleteChatRooms call site.
type deleteChatRoomsParams []struct {
	exchange uint16
	names    []string
	err      error
}

// chatRoomDeleterParams is a helper struct that
// contains mock parameters for ChatRoomDeleter methods.
type chatRoomDeleterParams struct {
	deleteChatRoomsParams
}

// chatSessionRetrieverAllSessionsParams is
// the list of parameters passed at the
// mock ChatSessionRetriever.AllSessions call site.
type chatSessionRetrieverAllSessionsParams []struct {
	cookie string
	result []*state.Session
}

// chatSessionRetrieverParams is a helper struct that
// contains mock parameters for ChatSessionRetriever methods.
type chatSessionRetrieverParams struct {
	chatSessionRetrieverAllSessionsParams
}

// broadcastVisibilityParams is the list of parameters passed at
// the mock BuddyBroadcaster.BroadcastVisibility call site.
type broadcastVisibilityParams []struct {
	filter         []state.IdentScreenName
	you            *state.SessionInstance
	err            error
	sendDepartures bool
}

// buddyBroadcasterParams is a helper struct that
// contains mock parameters for BuddyBroadcaster methods.
type buddyBroadcasterParams struct {
	broadcastVisibilityParams
}

// buddyIconMetadataParams is the list of parameters passed at the
// mock FeedBagRetriever.BuddyIconMetadataParams call site.
type buddyIconMetadataParams []struct {
	screenName state.IdentScreenName
	result     *wire.BARTID
	err        error
}

// allUsersParams is the list of parameters passed at
// the mock UserManager.AllUsers call site.
type allUsersParams []struct {
	result []state.User
	err    error
}

// getUserParams is the list of parameters passed at
// the mock UserManager.User call site.
type getUserParams []struct {
	screenName state.IdentScreenName
	result     *state.User
	err        error
}

// setUserPasswordParams is the list of parameters passed at
// the mock UserManager.SetUserPassword call site.
type setUserPasswordParams []struct {
	screenName  state.IdentScreenName
	newPassword string
	err         error
}

// insertUserParams is the list of parameters passed at
// the mock UserManager.InsertUser call site.
type insertUserParams []struct {
	u   state.User
	err error
}

// deleteUserParams is the list of parameters passed at
// the mock UserManager.DeleteUser call site.
type deleteUserParams []struct {
	screenName state.IdentScreenName
	err        error
}
