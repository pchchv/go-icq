package foodgroup

import (
	"log/slog"

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
