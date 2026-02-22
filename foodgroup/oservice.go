package foodgroup

import (
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
