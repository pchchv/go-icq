package foodgroup

import (
	"context"
	"log/slog"
	"time"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// FeedbagService provides functionality for the Feedbag food group,
// which handles buddy list management.
type FeedbagService struct {
	bartItemManager  BARTItemManager
	buddyBroadcaster buddyBroadcaster
	feedbagManager   FeedbagManager
	logger           *slog.Logger
	messageRelayer   MessageRelayer
}

// NewFeedbagService creates a new instance of FeedbagService.
func NewFeedbagService(
	logger *slog.Logger,
	messageRelayer MessageRelayer,
	feedbagManager FeedbagManager,
	bartItemManager BARTItemManager,
	relationshipFetcher RelationshipFetcher,
	sessionRetriever SessionRetriever,
) FeedbagService {
	return FeedbagService{
		bartItemManager:  bartItemManager,
		buddyBroadcaster: newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		feedbagManager:   feedbagManager,
		logger:           logger,
		messageRelayer:   messageRelayer,
	}
}

// StartCluster signals the beginning of a batch of feedbag operations that clients should
// process together to prevent UI flicker during rapid updates.
// It transmits the start message to other session instances.
func (s FeedbagService) StartCluster(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x13_0x11_FeedbagStartCluster) {
	s.messageRelayer.RelayToOtherInstances(ctx, instance, wire.SNACMessage{
		Frame: inFrame,
		Body:  inBody,
	})
}

// EndCluster signals the completion of a batched feedbag operation group.
// It transmits the end message to other session instances.
func (s FeedbagService) EndCluster(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame) {
	s.messageRelayer.RelayToOtherInstances(ctx, instance, wire.SNACMessage{
		Frame: inFrame,
		Body:  wire.SNAC_0x13_0x12_FeedbagEndCluster{},
	})
}

// Query fetches the user's feedbag (buddy list).
// It returns wire.FeedbagReply, which contains feedbag entries.
func (s FeedbagService) Query(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame) (wire.SNACMessage, error) {
	fb, err := s.feedbagManager.Feedbag(ctx, instance.IdentScreenName())
	if err != nil {
		return wire.SNACMessage{}, err
	}

	lm := time.UnixMilli(0)
	if len(fb) > 0 {
		lm, err = s.feedbagManager.FeedbagLastModified(ctx, instance.IdentScreenName())
		if err != nil {
			return wire.SNACMessage{}, err
		}
	}

	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Feedbag,
			SubGroup:  wire.FeedbagReply,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNAC_0x13_0x06_FeedbagReply{
			Version:    0,
			Items:      fb,
			LastUpdate: uint32(lm.Unix()),
		},
	}, nil
}

// QueryIfModified fetches the user's feedbag (aka buddy list).
// It returns wire.FeedbagReplyNotModified if the feedbag was last modified before inBody.LastUpdate,
// else return wire.FeedbagReply, which contains feedbag entries.
func (s FeedbagService) QueryIfModified(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x13_0x05_FeedbagQueryIfModified) (wire.SNACMessage, error) {
	fb, err := s.feedbagManager.Feedbag(ctx, instance.IdentScreenName())
	if err != nil {
		return wire.SNACMessage{}, err
	}

	lm := time.UnixMilli(0)
	if len(fb) > 0 {
		lm, err = s.feedbagManager.FeedbagLastModified(ctx, instance.IdentScreenName())
		if err != nil {
			return wire.SNACMessage{}, err
		} else if lm.Before(time.Unix(int64(inBody.LastUpdate), 0)) {
			return wire.SNACMessage{
				Frame: wire.SNACFrame{
					FoodGroup: wire.Feedbag,
					SubGroup:  wire.FeedbagReplyNotModified,
					RequestID: inFrame.RequestID,
				},
				Body: wire.SNAC_0x13_0x05_FeedbagQueryIfModified{
					LastUpdate: uint32(lm.Unix()),
					Count:      uint8(len(fb)),
				},
			}, nil
		}
	}

	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Feedbag,
			SubGroup:  wire.FeedbagReply,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNAC_0x13_0x06_FeedbagReply{
			Version:    0,
			Items:      fb,
			LastUpdate: uint32(lm.Unix()),
		},
	}, nil
}

// RightsQuery returns SNAC wire.FeedbagRightsReply,
// which contains Feedbag food group settings for the current user.
// The values within the SNAC are not well understood but seem to make the AIM client happy.
func (s FeedbagService) RightsQuery(_ context.Context, inFrame wire.SNACFrame) wire.SNACMessage {
	// maxItemsByClass defines per-type item limits
	// types not listed here are 0 by default
	// the slice size is equal to the maximum "enum" value+1
	maxItemsByClass := make([]uint16, 21)
	maxItemsByClass[wire.FeedbagClassIdBuddy] = 61
	maxItemsByClass[wire.FeedbagClassIdGroup] = 61
	maxItemsByClass[wire.FeedbagClassIDPermit] = 100
	maxItemsByClass[wire.FeedbagClassIDDeny] = 100
	maxItemsByClass[wire.FeedbagClassIdPdinfo] = 1
	maxItemsByClass[wire.FeedbagClassIdBuddyPrefs] = 1
	maxItemsByClass[wire.FeedbagClassIdNonbuddy] = 50
	maxItemsByClass[wire.FeedbagClassIdClientPrefs] = 3
	maxItemsByClass[wire.FeedbagClassIdWatchList] = 128
	maxItemsByClass[wire.FeedbagClassIdIgnoreList] = 255
	maxItemsByClass[wire.FeedbagClassIdDateTime] = 20
	maxItemsByClass[wire.FeedbagClassIdExternalUser] = 200
	maxItemsByClass[wire.FeedbagClassIdRootCreator] = 1
	maxItemsByClass[wire.FeedbagClassIdImportTimestamp] = 1
	maxItemsByClass[wire.FeedbagClassIdBart] = 200
	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Feedbag,
			SubGroup:  wire.FeedbagRightsReply,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNAC_0x13_0x03_FeedbagRightsReply{
			TLVRestBlock: wire.TLVRestBlock{
				TLVList: wire.TLVList{
					wire.NewTLVBE(wire.FeedbagRightsMaxItemAttrs, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsMaxItemsByClass, maxItemsByClass),
					wire.NewTLVBE(wire.FeedbagRightsMaxClientItems, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsMaxItemNameLen, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsMaxRecentBuddies, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsInteractionBuddies, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsInteractionHalfLife, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsInteractionMaxScore, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsMaxBuddiesPerGroup, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsMaxMegaBots, uint16(200)),
					wire.NewTLVBE(wire.FeedbagRightsMaxSmartGroups, uint16(100)),
				},
			},
		},
	}
}

// FeedbagBuddyPref returns a pref value stored in the user's feedbag.
//
// Preferences are binary values stored in a logical bitmask spanning 2 physical bitmasks.
// Each preference value is a position in the logical bitmask.
//
// The first bitmask (BuddyPrefs) is fixed-length of 32 bits (4 bytes).
// It's 0-offset: pref 1 is at offset 1, pref 2 at offset 2, etc.
// The most significant bit is on the right side.
//
// The second bitmask (BuddyPrefs2) is of an unbounded length.
// The values are at a position relative to the beginning at BuddyPrefs1.
// The most significant bit is on the left side.
//
// Items 1-31 are located in BuddyPrefs:
//
//	Item #1:
//	00000000 00000000 00000000 00000010 (BuddyPrefs)
//	                                 ^ offset 1, bit 2
//	00000000 00000000 00000000 00000000 (BuddyPrefs2)
//
//	Item #31:
//	10000000 00000000 00000000 00000000 (BuddyPrefs)
//	^ offset 31, bit 32
//	00000000 00000000 00000000 00000000 (BuddyPrefs2)
//
// Items 32+ are located in BuddyPrefs. To find the offset, calculate (Item #)-33.
// For example, item 52 is located at offset 19.
//
//	Item #52:
//	00000000 00000000 00000000 00000000 (BuddyPrefs)
//	00000000 00000000 00010000 00000000 (BuddyPrefs2)
//	                     ^ offset 19, bit 52
//
// There is a weird edge case for items 32 and 33 that is either a bug caused
// by the transition from offset to positional-based indexing,
// or a misunderstanding on my part: both items fall under offset 0 in BuddyPrefs2.
//
//	Item #32:
//	00000000 00000000 00000000 00000000 (BuddyPrefs)
//	10000000 00000000 00000000 00000000 (BuddyPrefs2)
//	^ offset 0, bit 33
//
//	Item #33:
//	00000000 00000000 00000000 00000000 (BuddyPrefs)
//	10000000 00000000 00000000 00000000 (BuddyPrefs2)
//	^ offset 0, bit 33
//
// For each logical bitmask, there are 2 physical bitmasks.
// The first contains the set values,
// and the second contains the valid bitmask positions.
// I guess this was done to remove ambiguity about an unset position:
// i.e. does an unset value mean false or null?
//
// The bitmasks are present in 4 TLVs:
//
// - 0x00C9: FeedbagAttributesBuddyPrefs
// - 0x00D6: FeedbagAttributesBuddyPrefsValid
// - 0x00D7: FeedbagAttributesBuddyPrefs2
// - 0x00D8: FeedbagAttributesBuddyPrefs2Valid
//
// For a given item, this function returns whether the preference number is
// available in the bitmask (valid) and what the value of it is (value).
func feedbagBuddyPref(prefNum uint16, list wire.TLVList) (valid bool, value bool) {
	offset := int(prefNum)
	// value is in BuddyPrefs; the most significant bit is on the right side
	if offset < 32 {
		buddyPrefValid, ok := list.Bytes(wire.FeedbagAttributesBuddyPrefsValid)
		if !ok {
			return false, false
		}

		buddyPrefEnabled, ok := list.Bytes(wire.FeedbagAttributesBuddyPrefs)
		if !ok {
			return false, false
		}

		index := (len(buddyPrefValid) - 1) - (offset / 8)
		if index >= len(buddyPrefValid) || index >= len(buddyPrefEnabled) {
			return false, false
		}

		bitOffset := offset % 8
		mask := byte(1 << bitOffset)
		valid = buddyPrefValid[index]&mask != 0
		value = buddyPrefEnabled[index]&mask != 0
		return
	}

	// value is in BuddyPrefs2
	// the most significant bit is on the left side
	if prefNum == 32 {
		offset = 0 // account for transition from offset-based to position-based
	} else {
		offset -= 33
	}

	buddyPrefValid, ok := list.Bytes(wire.FeedbagAttributesBuddyPrefs2Valid)
	if !ok {
		return false, false
	}

	buddyPrefEnabled, ok := list.Bytes(wire.FeedbagAttributesBuddyPrefs2)
	if !ok {
		return false, false
	}

	index := offset / 8
	if index >= len(buddyPrefValid) || index >= len(buddyPrefEnabled) {
		return false, false
	}

	bitOffset := offset % 8
	mask := byte(0x80) >> bitOffset
	valid = buddyPrefValid[index]&mask != 0
	value = buddyPrefEnabled[index]&mask != 0
	return
}
