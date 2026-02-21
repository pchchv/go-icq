package foodgroup

import (
	"context"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// PermitDenyService provides functionality for the PermitDeny (PD) food group.
// The PD food group manages settings for permit/deny (allow/block)
// for pre-feedbag (sever-side buddy list) AIM clients.
type PermitDenyService struct {
	buddyBroadcaster           buddyBroadcaster
	clientSideBuddyListManager ClientSideBuddyListManager
}

// NewPermitDenyService creates an instance of PermitDenyService.
func NewPermitDenyService(
	bartItemManager BARTItemManager,
	relationshipFetcher RelationshipFetcher,
	clientSideBuddyListManager ClientSideBuddyListManager,
	messageRelayer MessageRelayer,
	sessionRetriever SessionRetriever,
) PermitDenyService {
	return PermitDenyService{
		buddyBroadcaster:           newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		clientSideBuddyListManager: clientSideBuddyListManager,
	}
}

// AddDenyListEntries adds users to your block list and sets your visibility mode to "deny some".
// If your screen name is passed as a single element in the input payload,
// your visibility mode is set to "permit all" instead.
// Your buddy list and your relations' buddy lists are updated to reflect the current mode.
func (s PermitDenyService) AddDenyListEntries(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x09_0x07_PermitDenyAddDenyListEntries) error {
	if len(inBody.Users) == 1 {
		sn := state.NewIdentScreenName(inBody.Users[0].ScreenName)
		if sn.String() == instance.IdentScreenName().String() {
			if err := s.clientSideBuddyListManager.SetPDMode(ctx, instance.IdentScreenName(), wire.FeedbagPDModePermitAll); err != nil {
				return err
			}
			return s.maybeBroadcastVisibility(ctx, instance, nil)
		}
	}

	if err := s.clientSideBuddyListManager.SetPDMode(ctx, instance.IdentScreenName(), wire.FeedbagPDModeDenySome); err != nil {
		return err
	}

	for _, user := range inBody.Users {
		sn := state.NewIdentScreenName(user.ScreenName)
		if err := s.clientSideBuddyListManager.DenyBuddy(ctx, instance.IdentScreenName(), sn); err != nil {
			return err
		}
	}

	// don't filter users so that users permitted as
	// a result of this visibility change get properly notified
	return s.maybeBroadcastVisibility(ctx, instance, nil)
}

// AddPermListEntries adds users to your permit list and sets your visibility mode to "permit some".
// If your screen name is passed as a single element in the input payload,
// your visibility mode is set to "deny all" instead.
// Your buddy list and your relations' buddy lists are updated to reflect the current mode.
func (s PermitDenyService) AddPermListEntries(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x09_0x05_PermitDenyAddPermListEntries) error {
	if len(inBody.Users) == 1 {
		sn := state.NewIdentScreenName(inBody.Users[0].ScreenName)
		if sn.String() == instance.IdentScreenName().String() {
			if err := s.clientSideBuddyListManager.SetPDMode(ctx, instance.IdentScreenName(), wire.FeedbagPDModeDenyAll); err != nil {
				return err
			}
			return s.maybeBroadcastVisibility(ctx, instance, nil)
		}
	}

	if err := s.clientSideBuddyListManager.SetPDMode(ctx, instance.IdentScreenName(), wire.FeedbagPDModePermitSome); err != nil {
		return err
	}

	for _, user := range inBody.Users {
		sn := state.NewIdentScreenName(user.ScreenName)
		if err := s.clientSideBuddyListManager.PermitBuddy(ctx, instance.IdentScreenName(), sn); err != nil {
			return err
		}
	}

	// don't filter users so that users blocked as
	// a result of this visibility change get properly notified
	return s.maybeBroadcastVisibility(ctx, instance, nil)
}

// maybeBroadcastVisibility broadcasts visibility changes to a list users only if the client has finished signing in,
// which prevents duplicate arrival notifications, which are ultimately sent at the end of the sign on flow.
func (s PermitDenyService) maybeBroadcastVisibility(ctx context.Context, instance *state.SessionInstance, body []struct {
	ScreenName string `oscar:"len_prefix=uint8"`
}) error {
	if !instance.SignonComplete() {
		return nil
	}

	var filter []state.IdentScreenName
	if len(body) > 0 {
		filter = make([]state.IdentScreenName, 0, len(body))
		for _, user := range body {
			filter = append(filter, state.NewIdentScreenName(user.ScreenName))
		}
	}

	return s.buddyBroadcaster.BroadcastVisibility(ctx, instance, filter, true)
}
