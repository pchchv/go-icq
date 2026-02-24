package foodgroup

import (
	"context"
	"errors"
	"time"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// omitCaps is the map of to filter out of
// the client's capability list because they are
// not currently supported by the server.
var omitCaps = map[[16]byte]bool{
	wire.CapGames:      true, // games
	wire.CapSupportICQ: true, // ICQ inter-op
	wire.CapVoiceChat:  true, // voice chat
}

// LocateService provides functionality for the Locate food group,
// which is responsible for user profiles, user info lookups,
// directory information and keyword lookups.
type LocateService struct {
	buddyBroadcaster    buddyBroadcaster
	messageRelayer      MessageRelayer
	relationshipFetcher RelationshipFetcher
	profileManager      ProfileManager
	sessionRetriever    SessionRetriever
	userManager         UserManager
}

// NewLocateService creates a new instance of LocateService.
func NewLocateService(
	bartItemManager BARTItemManager,
	messageRelayer MessageRelayer,
	profileManager ProfileManager,
	relationshipFetcher RelationshipFetcher,
	sessionRetriever SessionRetriever,
	userManager UserManager,
) LocateService {
	return LocateService{
		buddyBroadcaster:    newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		messageRelayer:      messageRelayer,
		relationshipFetcher: relationshipFetcher,
		profileManager:      profileManager,
		sessionRetriever:    sessionRetriever,
		userManager:         userManager,
	}
}

// SetInfo sets the user's profile, away message or capabilities.
func (s LocateService) SetInfo(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x02_0x04_LocateSetInfo) error {
	// update profile
	if profileText, hasProfile := inBody.String(wire.LocateTLVTagsInfoSigData); hasProfile {
		mime, _ := inBody.String(wire.LocateTLVTagsInfoSigMime)
		profile := state.UserProfile{
			ProfileText: profileText,
			MIMEType:    mime,
			UpdateTime:  time.Now(),
		}

		// set the server-side profile
		if instance.KerberosAuth() || inBody.HasTag(wire.LocateTLVTagsInfoSupportHostSig) {
			// normally, the SupportHostSig TLV indicates that the profile should be stored server-side
			//
			// however, some AIM 6 clients expect server-side profiles but do not send this TLV
			//
			// in order to cover all bases, just save the profile for all kerberos-based clients
			if err := s.profileManager.SetProfile(ctx, instance.IdentScreenName(), profile); err != nil {
				return err
			}

			for _, _instance := range instance.Session().Instances() {
				if _instance.KerberosAuth() {
					// update all instances that do server-side profile storage
					_instance.SetProfile(profile)
				}
			}

			s.messageRelayer.RelayToOtherInstances(ctx, instance, wire.SNACMessage{
				Frame: wire.SNACFrame{
					FoodGroup: wire.OService,
					SubGroup:  wire.OServiceUserInfoUpdate,
				},
				Body: newOServiceUserInfoUpdate(instance),
			})
		} else {
			// set the client-side profile
			instance.SetProfile(profile)
		}
	}

	// broadcast away message change to buddies
	if awayMsg, hasAwayMsg := inBody.String(wire.LocateTLVTagsInfoUnavailableData); hasAwayMsg {
		if awayMsg != "" {
			instance.SetUserInfoFlag(wire.OServiceUserFlagUnavailable)
		} else {
			instance.ClearUserInfoFlag(wire.OServiceUserFlagUnavailable)
		}

		instance.SetAwayMessage(awayMsg)
		if instance.SignonComplete() {
			if err := s.buddyBroadcaster.BroadcastBuddyArrived(ctx, instance.IdentScreenName(), instance.Session().TLVUserInfo()); err != nil {
				return err
			}
		}
	}

	// update client capabilities (buddy icon, chat, etc...)
	if b, hasCaps := inBody.Bytes(wire.LocateTLVTagsInfoCapabilities); hasCaps {
		if len(b)%16 != 0 {
			return errors.New("capability list must be array of 16-byte values")
		}

		var caps [][16]byte
		for i := 0; i < len(b); i += 16 {
			var c [16]byte
			copy(c[:], b[i:i+16])
			if _, found := omitCaps[c]; !found {
				caps = append(caps, c)
			}
		}

		instance.SetCaps(caps)
		if instance.SignonComplete() {
			if err := s.buddyBroadcaster.BroadcastBuddyArrived(ctx, instance.IdentScreenName(), instance.Session().TLVUserInfo()); err != nil {
				return err
			}
		}
	}

	return nil
}
