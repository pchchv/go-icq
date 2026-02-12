package toc

import (
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

type addBuddiesParams []struct {
	me     state.IdentScreenName
	inBody wire.SNAC_0x03_0x04_BuddyAddBuddies
	err    error
}

type addDenyListEntriesParams []struct {
	me   state.IdentScreenName
	body wire.SNAC_0x09_0x07_PermitDenyAddDenyListEntries
	err  error
}

type addPermListEntriesParams []struct {
	me   state.IdentScreenName
	body wire.SNAC_0x09_0x05_PermitDenyAddPermListEntries
	err  error
}

type adminParams struct {
	infoChangeRequestParams
}

type authParams struct {
	crackCookieParams
	flapLoginParams
	signoutParams
	signoutChatParams
	registerBOSSessionParams
	registerChatSessionParams
}

type broadcastBuddyDepartedParams []struct {
	me  state.IdentScreenName
	err error
}

type channelMsgToHostParamsChat []struct {
	sender state.IdentScreenName
	inBody wire.SNAC_0x0E_0x05_ChatChannelMsgToHost
	result *wire.SNACMessage
	err    error
}

type channelMsgToHostParamsICBM []struct {
	sender  state.IdentScreenName
	inFrame wire.SNACFrame
	inBody  wire.SNAC_0x04_0x06_ICBMChannelMsgToHost
	result  *wire.SNACMessage
	err     error
}

type chatNavParams struct {
	createRoomParams
	requestRoomInfoParams
}

type chatParams struct {
	channelMsgToHostParamsChat
}

type clientOnlineParams []struct {
	body wire.SNAC_0x01_0x02_OServiceClientOnline
	me   state.IdentScreenName
	err  error
}

// cookieBakerParams groups the method scenarios for a CookieBaker.
type cookieBakerParams struct {
	issueParams issueParams
}

type crackCookieParams []struct {
	cookieIn  []byte
	cookieOut state.ServerCookie
	err       error
}

type createRoomParams []struct {
	me     state.IdentScreenName
	inBody wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate
	msg    wire.SNACMessage
	err    error
}

type delBuddiesParams []struct {
	me     state.IdentScreenName
	inBody wire.SNAC_0x03_0x05_BuddyDelBuddies
	err    error
}

type dirInfoParams []struct {
	body wire.SNAC_0x02_0x0B_LocateGetDirInfo
	msg  wire.SNACMessage
	err  error
}

type dirSearchParams struct {
	infoQueryParams
}

type evilRequestParams []struct {
	me     state.IdentScreenName
	inBody wire.SNAC_0x04_0x08_ICBMEvilRequest
	msg    wire.SNACMessage
	err    error
}

type flapLoginParams []struct {
	frame wire.FLAPSignonFrame
	tlv   wire.TLVRestBlock
	err   error
}

type icbmParams struct {
	channelMsgToHostParamsICBM
	evilRequestParams
}

type idleNotificationParams []struct {
	me     state.IdentScreenName
	bodyIn wire.SNAC_0x01_0x11_OServiceIdleNotification
	err    error
}

type infoChangeRequestParams []struct {
	me     state.IdentScreenName
	msg    wire.SNACMessage
	inBody wire.SNAC_0x07_0x04_AdminInfoChangeRequest
	err    error
}

type infoQueryParams []struct {
	inBody wire.SNAC_0x0F_0x02_InfoQuery
	msg    wire.SNACMessage
	err    error
}

// issueParams holds multiple scenarios for the Issue method.
type issueParams []struct {
	data       []byte
	returnData []byte
	returnErr  error
}

type oServiceParams struct {
	clientOnlineParams
	idleNotificationParams
	serviceRequestParams
}

type permitDenyParams struct {
	addDenyListEntriesParams
	addPermListEntriesParams
}

type registerBOSSessionParams []struct {
	authCookie state.ServerCookie
	instance   *state.SessionInstance
	err        error
}

type registerBuddyListParams []struct {
	user state.IdentScreenName
	err  error
}

type registerChatSessionParams []struct {
	authCookie state.ServerCookie
	instance   *state.SessionInstance
	err        error
}

type requestRoomInfoParams []struct {
	inBody wire.SNAC_0x0D_0x04_ChatNavRequestRoomInfo
	msg    wire.SNACMessage
	err    error
}

type retrieveSessionParams []struct {
	screenName      state.IdentScreenName
	returnedSession *state.Session
}

type serviceRequestParams []struct {
	me     state.IdentScreenName
	bodyIn wire.SNAC_0x01_0x04_OServiceServiceRequest
	msg    wire.SNACMessage
	err    error
}

type sessionRetrieverParams struct {
	retrieveSessionParams
}

type setDirInfoParams []struct {
	me     state.IdentScreenName
	inBody wire.SNAC_0x02_0x09_LocateSetDirInfo
	msg    wire.SNACMessage
	err    error
}

type setInfoParams []struct {
	me     state.IdentScreenName
	inBody wire.SNAC_0x02_0x04_LocateSetInfo
	err    error
}

type setTOCConfigParams []struct {
	user   state.IdentScreenName
	config string
	err    error
}

type signoutChatParams []struct {
	me state.IdentScreenName
}

type signoutParams []struct {
	me state.IdentScreenName
}

type unregisterBuddyListParams []struct {
	user state.IdentScreenName
	err  error
}
