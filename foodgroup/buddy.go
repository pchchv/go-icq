package foodgroup

import (
	"context"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// buddyNotifier centralizes logic for sending buddy arrival and departure notifications.
type buddyNotifier struct {
	bartItemManager     BARTItemManager
	relationshipFetcher RelationshipFetcher
	messageRelayer      MessageRelayer
	sessionRetriever    SessionRetriever
}

// unicastBuddyArrived sends the latest user info to a particular user.
// While updates are sent via the wire.BuddyArrived SNAC,
// the message is not only used to indicate the user coming online.
// It can also notify changes to buddy icons, warning levels, invisibility status, etc.
func (s buddyNotifier) unicastBuddyArrived(ctx context.Context, userInfo wire.TLVUserInfo, to state.IdentScreenName) {
	if !userInfo.IsInvisible() {
		s.messageRelayer.RelayToScreenName(ctx, to, wire.SNACMessage{
			Frame: wire.SNACFrame{
				FoodGroup: wire.Buddy,
				SubGroup:  wire.BuddyArrived,
				RequestID: wire.ReqIDFromServer,
			},
			Body: wire.SNAC_0x03_0x0B_BuddyArrived{
				TLVUserInfo: userInfo,
			},
		})
	}
}

func (s buddyNotifier) unicastBuddyDeparted(ctx context.Context, from *state.Session, to state.IdentScreenName) {
	s.messageRelayer.RelayToScreenName(ctx, to, wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Buddy,
			SubGroup:  wire.BuddyDeparted,
			RequestID: wire.ReqIDFromServer,
		},
		Body: wire.SNAC_0x03_0x0C_BuddyDeparted{
			TLVUserInfo: wire.TLVUserInfo{
				// don't include the TLV block, otherwise the AIM client fails to process the block event
				ScreenName:   from.IdentScreenName().String(),
				WarningLevel: from.Warning(),
			},
		},
	})
}
