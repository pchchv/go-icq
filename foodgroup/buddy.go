package foodgroup

import (
	"context"
	"fmt"

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

// BroadcastVisibility sends you and related users arrival/departure notifications
// that reflect your buddy list and privacy preferences.
//
// Behavior:
//   - Sends you arrival notifications for users on your buddy list that I do not block.
//   - Sends arrival notifications to users that you block who have you on their buddy lists.
//   - Sends you departure notifications for users on your buddy list that you block  (if doSendDepartures is true).
//   - Sends departure notifications to users that you block who have you on their buddy lists (if doSendDepartures is true).
//   - Don't send notifications for any user that blocks you.
//
// This method is called when your visibility settings change,
// ensuring that all relevant users are notified of your arrival or departure status.
func (s buddyNotifier) BroadcastVisibility(ctx context.Context, you *state.SessionInstance, filter []state.IdentScreenName, doSendDepartures bool) error {
	relationships, err := s.relationshipFetcher.AllRelationships(ctx, you.IdentScreenName(), filter)
	if err != nil {
		return fmt.Errorf("retrieving relationships: %w", err)
	}

	yourTLVInfo := you.Session().TLVUserInfo()
	for _, relationship := range relationships {
		if relationship.BlocksYou {
			continue // they block you, don't send them notifications
		}

		theirSess := s.sessionRetriever.RetrieveSession(relationship.User)
		if theirSess == nil {
			continue // they are offline
		}

		if !relationship.YouBlock {
			if relationship.IsOnTheirList {
				// tell them you're online
				s.unicastBuddyArrived(ctx, yourTLVInfo, theirSess.IdentScreenName())
			}

			if relationship.IsOnYourList {
				theirInfo := theirSess.TLVUserInfo()
				// tell you they're online
				s.unicastBuddyArrived(ctx, theirInfo, you.IdentScreenName())
			}
		} else if relationship.YouBlock && doSendDepartures {
			if relationship.IsOnTheirList {
				// tell them you're offline
				s.unicastBuddyDeparted(ctx, you.Session(), theirSess.IdentScreenName())
			}

			if relationship.IsOnYourList {
				// tell you they're offline
				s.unicastBuddyDeparted(ctx, theirSess, you.IdentScreenName())
			}
		}
	}

	return nil
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
