package foodgroup

import (
	"net/mail"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// accountManagerConfirmStatusParams is the list of parameters passed at
// the mock accountManager.ConfirmStatus call site.
type accountManagerConfirmStatusParams []struct {
	screenName    state.IdentScreenName
	confirmStatus bool
	err           error
}

// accountManagerEmailAddressParams is the list of parameters passed at
// the mock accountManager.EmailAddress call site.
type accountManagerEmailAddressParams []struct {
	screenName   state.IdentScreenName
	emailAddress *mail.Address
	err          error
}

// accountManagerParams is a helper struct that
// contains mock parameters for accountManager methods.
type accountManagerParams struct {
	accountManagerUpdateDisplayScreenNameParams
	accountManagerUpdateEmailAddressParams
	accountManagerEmailAddressParams
	accountManagerUpdateRegStatusParams
	accountManagerRegStatusParams
	accountManagerUpdateConfirmStatusParams
	accountManagerConfirmStatusParams
	accountManagerUserParams
	accountManagerSetUserPasswordParams
}

// accountManagerRegStatusParams is the list of parameters passed at
// the mock accountManager.RegStatus call site.
type accountManagerRegStatusParams []struct {
	screenName state.IdentScreenName
	regStatus  uint16
	err        error
}

// accountManagerSetUserPasswordParams is the list of parameters passed at
// the mock accountManager.SetUserPassword call site.
type accountManagerSetUserPasswordParams []struct {
	screenName state.IdentScreenName
	password   string
	err        error
}

// accountManagerUpdateConfirmStatusParams is the list of parameters passed at
// the mock accountManager.UpdateConfirmStatus call site.
type accountManagerUpdateConfirmStatusParams []struct {
	confirmStatus bool
	screenName    state.IdentScreenName
	err           error
}

// accountManagerUpdateDisplayScreenNameParams is the list of parameters passed at
// the mock accountManager.UpdateDisplayScreenName call site.
type accountManagerUpdateDisplayScreenNameParams []struct {
	displayScreenName state.DisplayScreenName
	err               error
}

// accountManagerUpdateEmailAddressParams is the list of parameters passed at
// the mock accountManager.UpdateEmailAddress call site.
type accountManagerUpdateEmailAddressParams []struct {
	emailAddress *mail.Address
	screenName   state.IdentScreenName
	err          error
}

// accountManagerUpdateRegStatusParams is the list of parameters passed at
// the mock accountManager.UpdateRegStatus call site.
type accountManagerUpdateRegStatusParams []struct {
	regStatus  uint16
	screenName state.IdentScreenName
	err        error
}

// accountManagerUserParams is the list of parameters passed at
// the mock accountManager.User call site.
type accountManagerUserParams []struct {
	screenName state.IdentScreenName
	result     *state.User
	err        error
}

// legacyBuddiesParams is the list of parameters passed at
// the mock ClientSideBuddyListManager.AddBuddy call site.
type addBuddyParams []struct {
	me   state.IdentScreenName
	them state.IdentScreenName
	err  error
}

// addSessionParams is the list of parameters passed at
// the mock SessionRegistry.AddSession call site.
type addSessionParams []struct {
	screenName  state.DisplayScreenName
	doMultiSess bool
	result      *state.SessionInstance
	err         error
}

// adjacentUsersParams is the list of parameters passed at
// the mock FeedbagManager.AdjacentUsers call site.
type adjacentUsersParams []struct {
	screenName state.IdentScreenName
	users      []state.IdentScreenName
	err        error
}

// allRelationshipsParams is the list of parameters passed at
// the mock RelationshipFetcher.AllRelationships call site.
type allRelationshipsParams []struct {
	screenName state.IdentScreenName
	filter     []state.IdentScreenName
	result     []state.Relationship
	err        error
}

// broadcastBuddyArrivedParams is the list of parameters passed at
// the mock buddyBroadcaster.BroadcastBuddyArrived call site.
type broadcastBuddyArrivedParams []struct {
	screenName  state.DisplayScreenName
	err         error
	bodyMatcher func(snac wire.TLVUserInfo) bool
}

// broadcastBuddyDepartedParams is the list of parameters passed at
// the mock buddyBroadcaster.BroadcastBuddyDeparted call site.
type broadcastBuddyDepartedParams []struct {
	screenName state.IdentScreenName
	err        error
}

// broadcastVisibilityParams is the list of parameters passed at
// the mock buddyBroadcaster.BroadcastVisibility call site.
type broadcastVisibilityParams []struct {
	from             state.IdentScreenName
	filter           []state.IdentScreenName
	doSendDepartures bool
	err              error
}

// buddiesParams is the list of parameters passed at
// the mock FeedbagManager.Buddies call site.
type buddiesParams []struct {
	screenName state.IdentScreenName
	results    []state.IdentScreenName
}

// buddyBroadcasterParams is a helper struct that
// contains mock parameters for buddyBroadcaster methods.
type buddyBroadcasterParams struct {
	broadcastBuddyArrivedParams
	broadcastBuddyDepartedParams
	broadcastVisibilityParams
}

// buddyIconMetadataParams is the list of parameters passed at
// the mock BARTItemManager.BuddyIconMetadata call site.
type buddyIconMetadataParams []struct {
	screenName state.IdentScreenName
	result     *wire.BARTID
	err        error
}
