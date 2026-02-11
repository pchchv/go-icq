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
