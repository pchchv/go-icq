package state

import "github.com/pchchv/go-icq/wire"

func newFeedbagItem(classID uint16, itemID uint16, name string) wire.FeedbagItem {
	return wire.FeedbagItem{
		ClassID: classID,
		ItemID:  itemID,
		Name:    name,
	}
}
