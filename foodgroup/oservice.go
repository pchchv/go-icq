package foodgroup

import (
	"context"
	"log/slog"
	"time"

	"github.com/pchchv/go-icq/config"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// OServiceService provides functionality for the OService food group,
// which provides an assortment of services useful across multiple food groups.
type OServiceService struct {
	buddyBroadcaster      buddyBroadcaster
	cfg                   config.Config // todo remove
	logger                *slog.Logger
	snacRateLimits        wire.SNACRateLimits
	timeNow               func() time.Time
	chatRoomManager       ChatRoomRegistry
	cookieIssuer          CookieBaker
	messageRelayer        MessageRelayer
	chatMessageRelayer    ChatMessageRelayer
	profileManager        ProfileManager
	offlineMessageManager OfflineMessageManager
}

// NewOServiceService creates a new instance of NewOServiceService.
func NewOServiceService(
	cfg config.Config,
	messageRelayer MessageRelayer,
	logger *slog.Logger,
	cookieIssuer CookieBaker,
	chatRoomManager ChatRoomRegistry,
	relationshipFetcher RelationshipFetcher,
	sessionRetriever SessionRetriever,
	bartItemManager BARTItemManager,
	snacRateLimits wire.SNACRateLimits,
	chatMessageRelayer ChatMessageRelayer,
	profileManager ProfileManager,
	offlineMessageManager OfflineMessageManager,
) *OServiceService {
	return &OServiceService{
		cookieIssuer:          cookieIssuer,
		messageRelayer:        messageRelayer,
		buddyBroadcaster:      newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		cfg:                   cfg,
		logger:                logger,
		snacRateLimits:        snacRateLimits,
		timeNow:               time.Now,
		chatRoomManager:       chatRoomManager,
		chatMessageRelayer:    chatMessageRelayer,
		profileManager:        profileManager,
		offlineMessageManager: offlineMessageManager,
	}
}

// UserInfoQuery returns SNAC wire.OServiceUserInfoUpdate containing the user's info.
func (s OServiceService) UserInfoQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame) wire.SNACMessage {
	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.OService,
			SubGroup:  wire.OServiceUserInfoUpdate,
			RequestID: inFrame.RequestID,
		},
		Body: newOServiceUserInfoUpdate(instance),
	}
}

// SetUserInfoFields updates user info fields (e.g., invisible, away) and
// broadcasts presence changes to buddies.
// Returns an updated user info message.
func (s OServiceService) SetUserInfoFields(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x01_0x1E_OServiceSetUserInfoFields) (wire.SNACMessage, error) {
	if status, hasStatus := inBody.Uint32BE(wire.OServiceUserInfoStatus); hasStatus {
		instance.SetUserStatusBitmask(status)
		if instance.Session().Invisible() {
			if err := s.buddyBroadcaster.BroadcastBuddyDeparted(ctx, instance); err != nil {
				return wire.SNACMessage{}, err
			}
		} else {
			if err := s.buddyBroadcaster.BroadcastBuddyArrived(ctx, instance.IdentScreenName(), instance.Session().TLVUserInfo()); err != nil {
				return wire.SNACMessage{}, err
			}
		}
	}

	// reflect the status of this instance back to the caller,
	// even though it does not reflect aggregated state of the session
	//
	// this is necessary for the "invisible" button to properly toggle on the client
	info := instance.Session().TLVUserInfo()
	info.Replace(wire.NewTLVBE(wire.OServiceUserInfoStatus, instance.UserStatusBitmask()))
	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.OService,
			SubGroup:  wire.OServiceUserInfoUpdate,
			RequestID: inFrame.RequestID,
		},
		Body: wire.SNAC_0x01_0x0F_OServiceUserInfoUpdate{
			UserInfo: []wire.TLVUserInfo{info},
		},
	}, nil
}

// SetPrivacyFlags sets client privacy settings.
// Currently, there's no action to take when these flags are set.
// This method simply logs the flags set by the client.
func (s OServiceService) SetPrivacyFlags(ctx context.Context, inBody wire.SNAC_0x01_0x14_OServiceSetPrivacyFlags) {
	attrs := slog.Group("request", slog.String("food_group", wire.FoodGroupName(wire.OService)), slog.String("sub_group", wire.SubGroupName(wire.OService, wire.OServiceSetPrivacyFlags)))
	if inBody.MemberFlag() {
		s.logger.LogAttrs(ctx, slog.LevelDebug, "client set member privacy flag, but we're not going to do anything", attrs)
	}
	if inBody.IdleFlag() {
		s.logger.LogAttrs(ctx, slog.LevelDebug, "client set idle privacy flag, but we're not going to do anything", attrs)
	}
}

// IdleNotification sets the user idle time.
// Set session idle time to the value of bodyIn.IdleTime.
// Return a user arrival message to all users who have this user on their buddy list.
func (s OServiceService) IdleNotification(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x01_0x11_OServiceIdleNotification) error {
	if inBody.IdleTime == 0 {
		instance.UnsetIdle()
	} else {
		instance.SetIdle(time.Duration(inBody.IdleTime) * time.Second)
	}

	return s.buddyBroadcaster.BroadcastBuddyArrived(ctx, instance.IdentScreenName(), instance.Session().TLVUserInfo())
}

// newOServiceUserInfoUpdate constructs SNAC(0x01,0x0F) for user info updates.
// For OService version 4 and above, it appends a duplicate TLVUserInfo block.
// AIM 6+ expects at least two user info blocks to support multi-session:
// the first represents overall state; subsequent ones represent client instances.
func newOServiceUserInfoUpdate(instance *state.SessionInstance) wire.SNAC_0x01_0x0F_OServiceUserInfoUpdate {
	info := instance.Session().TLVUserInfo()
	userInfo := []wire.TLVUserInfo{info}
	// set registration date
	userInfo[0].Append(wire.NewTLVBE(wire.OServiceUserInfoMemberSince, uint32(instance.Session().MemberSince().Unix())))
	// set sign-on time
	userInfo[0].Append(wire.NewTLVBE(wire.OServiceUserInfoSignonTOD, uint32(instance.SignonTime().Unix())))
	// set current session length (seconds)
	userInfo[0].Append(wire.NewTLVBE(wire.OServiceUserInfoOnlineTime, uint32(time.Since(instance.SignonTime()).Seconds())))
	if instance.FoodGroupVersions()[wire.OService] >= 4 {
		userInfo[0].Append(wire.NewTLVBE(wire.OServiceUserInfoMyInstanceNum, []byte{instance.Num()}))
		for _, instance := range instance.Session().Instances() {
			instanceInfo := wire.TLVUserInfo{
				ScreenName:   instance.DisplayScreenName().String(),
				WarningLevel: instance.Warning(),
			}

			// sign-in timestamp
			instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoSignonTOD, uint32(instance.SignonTime().Unix())))
			// use the first instance as a template
			uFlags := instance.UserInfoBitmask()
			if instance.Session().Away() {
				uFlags |= wire.OServiceUserFlagUnavailable
			}

			instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoUserFlags, uFlags))
			// user status flags - user-level (shared)
			var statusBitmask uint32
			if instance.Invisible() {
				statusBitmask |= wire.OServiceUserStatusInvisible
			}

			instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoStatus, statusBitmask))
			if icon, hasIcon := instance.Session().BuddyIcon(); hasIcon {
				// set buddy icon metadata, if user has buddy icon
				if icon.Type != 0 {
					instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoBARTInfo, icon))
				}
			}

			// get the best instance for each TLV value
			// mostCapableCaps := instance.getMostCapableCaps()
			// capabilities - show most capable instance (union of all capabilities)
			instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoOscarCaps, instance.Session().Caps()))
			instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoMySubscriptions, uint32(0)))
			profile := instance.Profile()
			if !profile.UpdateTime.IsZero() {
				// set profile update time if the profile was set
				instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoSigTime, uint32(profile.UpdateTime.Unix())))
			}

			instanceInfo.Append(wire.NewTLVBE(wire.OServiceUserInfoPrimaryInstance, []byte{instance.Num()}))
			userInfo = append(userInfo, instanceInfo)
		}
	}

	return wire.SNAC_0x01_0x0F_OServiceUserInfoUpdate{
		UserInfo: userInfo,
	}
}
