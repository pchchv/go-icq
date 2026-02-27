package foodgroup

import (
	"net/mail"
	"time"

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

// bartItemManagerParams is a helper struct that
// contains mock parameters for BARTItemManager methods.
type bartItemManagerParams struct {
	bartItemManagerRetrieveParams
	bartItemManagerUpsertParams
	buddyIconMetadataParams
}

// bartItemManagerRetrieveParams is the list of parameters passed at
// the mock BARTItemManager.BuddyIcon call site.
type bartItemManagerRetrieveParams []struct {
	itemHash []byte
	result   []byte
	err      error
}

// bartItemManagerUpsertParams is the list of parameters passed at
// the mock BARTItemManager.SetBuddyIcon call site.
type bartItemManagerUpsertParams []struct {
	itemHash []byte
	payload  []byte
	bartType uint16
	err      error
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

// chatAllSessionsParams is the list of parameters passed at
// the mock ChatMessageRelayer.AllSessions call site.
type chatAllSessionsParams []struct {
	cookie   string
	sessions []*state.Session
	err      error
}

// chatMessageRelayerParams is a helper struct that
// contains mock parameters for ChatMessageRelayer methods.
type chatMessageRelayerParams struct {
	chatAllSessionsParams
	chatRelayToAllExceptParams
	chatRelayToScreenNameParams
}

// chatRelayToAllExceptParams is the list of parameters passed at
// the mock ChatMessageRelayer.RelayToAllExcept call site.
type chatRelayToAllExceptParams []struct {
	cookie     string
	screenName state.IdentScreenName
	message    wire.SNACMessage
	err        error
}

// chatRelayToScreenNameParams is the list of parameters passed at
// the mock ChatMessageRelayer.RelayToScreenName call site.
type chatRelayToScreenNameParams []struct {
	cookie     string
	screenName state.IdentScreenName
	message    wire.SNACMessage
	err        error
}

// chatRoomByCookieParams is the list of parameters passed at
// the mock ChatRoomRegistry.ChatRoomByCookie call site.
type chatRoomByCookieParams []struct {
	cookie string
	room   state.ChatRoom
	err    error
}

// chatRoomByCookieParams is the list of parameters passed at
// the mock ChatRoomRegistry.ChatRoomByName call site.
type chatRoomByNameParams []struct {
	exchange uint16
	name     string
	room     state.ChatRoom
	err      error
}

// chatRoomRegistryParams is a helper struct that
// contains mock parameters for ChatRoomRegistry methods.
type chatRoomRegistryParams struct {
	chatRoomByCookieParams
	chatRoomByNameParams
	createChatRoomParams
}

// cookieBakerParams is a helper struct that
// contains mock parameters for CookieBaker methods.
type cookieBakerParams struct {
	cookieCrackParams
	cookieIssueParams
}

// cookieCrackParams is the list of parameters passed at
// the mock CookieBaker.Crack call site.
type cookieCrackParams []struct {
	cookieIn []byte
	dataOut  []byte
	err      error
}

// cookieIssueParams is the list of parameters passed at
// the mock CookieBaker.Issue call site.
type cookieIssueParams []struct {
	dataIn    []byte
	cookieOut []byte
	err       error
}

// createChatRoomParams is the list of parameters passed at
// the mock ChatRoomRegistry.CreateChatRoom call site.
type createChatRoomParams []struct {
	room *state.ChatRoom
	err  error
}

// legacyBuddiesParams is the list of parameters passed at
// the mock ClientSideBuddyListManager.RemoveBuddy call site.
type deleteBuddyParams []struct {
	me   state.IdentScreenName
	them state.IdentScreenName
	err  error
}

// deleteMessagesParams is the list of parameters passed at
// the mock OfflineMessageManager.DeleteMessages call site.
type deleteMessagesParams []struct {
	recipIn state.IdentScreenName
	err     error
}

// deleteUserParams is the list of parameters passed at
// the mock ClientSideBuddyListManager.RemoveBuddy call site.
type denyBuddyParams []struct {
	me   state.IdentScreenName
	them state.IdentScreenName
	err  error
}

// findByAIMEmailParams is the list of parameters passed at
// the mock ProfileManager.FindByAIMEmail call site.
type findByAIMEmailParams []struct {
	email  string
	result state.User
	err    error
}

// findByAIMKeywordParams is the list of parameters passed at
// the mock ProfileManager.FindByAIMKeyword call site.
type findByAIMKeywordParams []struct {
	keyword string
	result  []state.User
	err     error
}

// findByAIMNameAndAddrParams is the list of parameters passed at
// the mock ProfileManager.FindByAIMNameAndAddr call site.
type findByAIMNameAndAddrParams []struct {
	info   state.AIMNameAndAddr
	result []state.User
	err    error
}

// setBasicInfoParams is the list of parameters passed at
// the mock ICQUserFinder.FindByDetails call site.
type findByDetailsParams []struct {
	firstName string
	lastName  string
	nickName  string
	result    []state.User
	err       error
}

// findByEmailParams is the list of parameters passed at
// the mock ICQUserFinder.FindByEmail call site.
type findByEmailParams []struct {
	email  string
	result state.User
	err    error
}

// setBasicInfoParams is the list of parameters passed at
// the mock ICQUserFinder.FindByInterests call site.
type findByInterestsParams []struct {
	code     uint16
	keywords []string
	result   []state.User
	err      error
}

// findByKeywordParams is the list of parameters passed at
// the mock ICQUserFinder.FindByKeyword call site.
type findByKeywordParams []struct {
	keyword string
	result  []state.User
	err     error
}

// findByUINParams is the list of parameters passed at
// the mock ICQUserFinder.FindByUIN call site.
type findByUINParams []struct {
	UIN    uint32
	result state.User
	err    error
}

// feedbagDeleteParams is the list of parameters passed at
// the mock FeedbagManager.FeedbagDelete call site.
type feedbagDeleteParams []struct {
	screenName state.IdentScreenName
	items      []wire.FeedbagItem
}

// feedbagLastModifiedParams is the list of parameters passed at
// the mock FeedbagManager.FeedbagLastModified call site.
type feedbagLastModifiedParams []struct {
	screenName state.IdentScreenName
	result     time.Time
}

// feedbagParams is the list of parameters passed at
// the mock FeedbagManager.Feedbag call site.
type feedbagParams []struct {
	screenName state.IdentScreenName
	results    []wire.FeedbagItem
	err        error
}

// feedbagUpsertParams is the list of parameters passed at
// the mock FeedbagManager.FeedbagUpsert call site.
type feedbagUpsertParams []struct {
	screenName state.IdentScreenName
	items      []wire.FeedbagItem
}

// getUserParams is the list of parameters passed at
// the mock UserManager.User call site.
type getUserParams []struct {
	screenName state.IdentScreenName
	result     *state.User
	err        error
}

// icqUserFinderParams is a helper struct that
// contains mock parameters for ICQUserFinder methods.
type icqUserFinderParams struct {
	findByDetailsParams
	findByEmailParams
	findByInterestsParams
	findByKeywordParams
	findByUINParams
}

// interestListParams is the list of parameters passed at
// the mock ProfileManager.InterestList call site.
type interestListParams []struct {
	result []wire.ODirKeywordListItem
	err    error
}

// messageRelayerParams is a helper struct that
// contains mock parameters for MessageRelayer methods.
type messageRelayerParams struct {
	relayToScreenNamesParams
	relayToScreenNameParams
	relayToOtherInstancesParams
	relayToScreenNameActiveOnlyParams
	relayToSelfParams
}

// permitBuddyParams is the list of parameters passed at
// the mock ClientSideBuddyListManager.PermitBuddy call site.
type permitBuddyParams []struct {
	me   state.IdentScreenName
	them state.IdentScreenName
	err  error
}

// relationshipFetcherParams is a helper struct that
// contains mock parameters for RelationshipFetcher methods.
type relationshipFetcherParams struct {
	allRelationshipsParams
	relationshipParams
}

// relationshipParams is the list of parameters passed at
// the mock RelationshipFetcher.Relationship call site.
type relationshipParams []struct {
	me     state.IdentScreenName
	them   state.IdentScreenName
	result state.Relationship
	err    error
}

// relayToOtherInstancesParams is the list of parameters passed at
// the mock MessageRelayer.RelayToOtherInstances call site.
type relayToOtherInstancesParams []struct {
	screenName state.IdentScreenName
	message    wire.SNACMessage
}

// relayToScreenNameActiveOnlyParams is the list of parameters passed at
// the mock MessageRelayer.RelayToScreenNameActiveOnly call site.
type relayToScreenNameActiveOnlyParams []struct {
	screenName state.IdentScreenName
	message    wire.SNACMessage
}

// relayToScreenNameParams is the list of parameters passed at
// the mock MessageRelayer.RelayToScreenName call site.
type relayToScreenNameParams []struct {
	screenName state.IdentScreenName
	message    wire.SNACMessage
}

// relayToScreenNamesParams is the list of parameters passed at
// the mock MessageRelayer.RelayToScreenNames call site.
type relayToScreenNamesParams []struct {
	screenNames []state.IdentScreenName
	message     wire.SNACMessage
}

// relayToSelfParams is the list of parameters passed at
// the mock MessageRelayer.RelayToSelf call site.
type relayToSelfParams []struct {
	screenName state.IdentScreenName
	message    wire.SNACMessage
}

// removeDenyBuddyParams is the list of parameters passed at
// the mock ClientSideBuddyListManager.RemoveDenyBuddy call site.
type removeDenyBuddyParams []struct {
	me   state.IdentScreenName
	them state.IdentScreenName
	err  error
}

// // removePermitBuddyParams is the list of parameters passed at
// the mock ClientSideBuddyListManager.RemovePermitBuddy call site.
type removePermitBuddyParams []struct {
	me   state.IdentScreenName
	them state.IdentScreenName
	err  error
}

// removeSessionParams is the list of parameters passed at
// the mock SessionRegistry.RemoveSession call site.
type removeSessionParams []struct {
	screenName state.IdentScreenName
}

// deleteMessagesParams is the list of parameters passed at
// the mock OfflineMessageManager.RetrieveMessages call site.
type retrieveMessagesParams []struct {
	recipIn     state.IdentScreenName
	messagesOut []state.OfflineMessage
	err         error
}

// retrieveProfileParams is the list of parameters passed at
// the mock ProfileManager.Profile call site.
type retrieveProfileParams []struct {
	screenName state.IdentScreenName
	result     state.UserProfile
	err        error
}

// retrieveSessionParams is the list of parameters passed at
// the mock SessionRetriever.RetrieveSession call site.
type retrieveSessionParams []struct {
	screenName state.IdentScreenName
	result     *state.Session
}

// useParams is the list of parameters passed at
// the mock FeedbagManager.Use call site.
type useParams []struct {
	screenName state.IdentScreenName
}

// userManagerParams is a helper struct that
// contains mock parameters for UserManager methods.
type userManagerParams struct {
	getUserParams
}
