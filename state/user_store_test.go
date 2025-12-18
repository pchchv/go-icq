package state

import "github.com/pchchv/go-icq/wire"

func newFeedbagItem(classID uint16, itemID uint16, name string) wire.FeedbagItem {
	return wire.FeedbagItem{
		ClassID: classID,
		ItemID:  itemID,
		Name:    name,
	}
}

func pdInfoItem(itemID uint16, pdMode wire.FeedbagPDMode) wire.FeedbagItem {
	return wire.FeedbagItem{
		ClassID: wire.FeedbagClassIdPdinfo,
		ItemID:  itemID,
		TLVLBlock: wire.TLVLBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.FeedbagAttributesPdMode, uint8(pdMode)),
			},
		},
	}
}
