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

type chatParams struct {
	channelMsgToHostParamsChat
}

type clientOnlineParams []struct {
	body wire.SNAC_0x01_0x02_OServiceClientOnline
	me   state.IdentScreenName
	err  error
}

type createRoomParams []struct {
	me     state.IdentScreenName
	inBody wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate
	msg    wire.SNACMessage
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
