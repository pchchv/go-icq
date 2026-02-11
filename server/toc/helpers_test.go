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
