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

// RateParamsQuery returns SNAC rate limits.
// It returns SNAC wire.OServiceRateParamsReply containing rate limits for all food groups supported by this server.
//
// The purpose of this method is to convey per-SNAC server-side rate limits to the client.
// The response consists of two main parts: rate classes and rate groups.
// Rate classes define limits based on specific parameters,
// while rate groups associate these limits with relevant SNAC types.
//
// The current implementation does not enforce server-side rate limiting.
// Instead, the provided values inform the client about the recommended client-side rate limits.
//
// AIM clients silently fail when they expect a rate limit rule that does not exist in this response.
// When support for a new food group is added to the server, update this function accordingly.
func (s OServiceService) RateParamsQuery(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame) wire.SNACMessage {
	// not contain LastTime and CurrentStatus fields.
	var limits = wire.SNAC_0x01_0x07_OServiceRateParamsReply{
		RateClasses: []wire.RateParamsSNAC{},
		RateGroups: []struct {
			ID    uint16
			Pairs []struct {
				FoodGroup uint16
				SubGroup  uint16
			} `oscar:"count_prefix=uint16"`
		}{
			{
				ID: 1,
				Pairs: []struct {
					FoodGroup uint16
					SubGroup  uint16
				}{},
			},
			{
				ID: 2,
				Pairs: []struct {
					FoodGroup uint16
					SubGroup  uint16
				}{},
			},
			{
				ID: 3,
				Pairs: []struct {
					FoodGroup uint16
					SubGroup  uint16
				}{},
			},
			{
				ID: 4,
				Pairs: []struct {
					FoodGroup uint16
					SubGroup  uint16
				}{},
			},
			{
				ID: 5,
				Pairs: []struct {
					FoodGroup uint16
					SubGroup  uint16
				}{},
			},
		},
	}

	for _, class := range instance.RateLimitStates() {
		str := wire.RateParamsSNAC{
			ID:              uint16(class.ID),
			WindowSize:      uint32(class.WindowSize),
			ClearLevel:      uint32(class.ClearLevel),
			AlertLevel:      uint32(class.AlertLevel),
			LimitLevel:      uint32(class.LimitLevel),
			DisconnectLevel: uint32(class.DisconnectLevel),
			CurrentLevel:    uint32(class.CurrentLevel),
			MaxLevel:        uint32(class.MaxLevel),
		}
		if instance.FoodGroupVersions()[wire.OService] > 1 {
			str.V2Params = &struct {
				LastTime      uint32
				DroppingSNACs uint8
			}{
				LastTime: uint32(s.timeNow().Add(-time.Second).Unix()),
			}
		}
		limits.RateClasses = append(limits.RateClasses, str)
	}

	for snacClass := range s.snacRateLimits.All() {
		classID := int(snacClass.RateLimitClass) - 1
		limits.RateGroups[classID].Pairs = append(limits.RateGroups[classID].Pairs,
			struct {
				FoodGroup uint16
				SubGroup  uint16
			}{FoodGroup: snacClass.FoodGroup, SubGroup: snacClass.SubGroup})
	}

	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.OService,
			SubGroup:  wire.OServiceRateParamsReply,
			RequestID: inFrame.RequestID,
		},
		Body: limits,
	}
}

// RateParamsSubAdd subscribes to rate parameter changes.
// AOL's OSCAR spec says that notifications will be queued after calling this method.
// I don't see the point of doing that since all clients appear to call RateParamsQuery at sign-on for all rate classes.
func (s OServiceService) RateParamsSubAdd(ctx context.Context, instance *state.SessionInstance, inBody wire.SNAC_0x01_0x08_OServiceRateParamsSubAdd) {
	ids := make([]wire.RateLimitClassID, 0, len(inBody.ClassIDs))
	for _, id := range inBody.ClassIDs {
		if id < 1 || id > 5 {
			s.logger.DebugContext(ctx, "snac class ID out of range")
			continue
		}

		ids = append(ids, wire.RateLimitClassID(id))
	}

	if len(ids) == 0 {
		return
	}

	s.logger.DebugContext(ctx, "subscribing to rate limit updates", "classes", ids)
	instance.Session().SubscribeRateLimits(ids)
}

// RateLimitUpdates produces update messages reflecting any recent changes in
// rate limit class params or rate limit states for the current session.
// Changes are reported relative to the previous invocation for this session.
// Only newly observed transitions or updated rate parameters will be included.
func (s OServiceService) RateLimitUpdates(ctx context.Context, instance *state.SessionInstance, now time.Time) []wire.SNACMessage {
	msgs := make([]wire.SNACMessage, 0, 5)
	classDelta, stateDelta := instance.Session().ObserveRateChanges(now)

	for _, curRate := range classDelta {
		s.logger.DebugContext(ctx, "rate limit class changed", "class", curRate.ID)
		msgs = append(msgs, buildRateLimitUpdate(1, curRate, instance, now))
	}

	for _, curRate := range stateDelta {
		s.logger.DebugContext(ctx, "rate limit state changed",
			"class", curRate.ID,
			"state", curRate.CurrentStatus)
		var code uint16
		switch curRate.CurrentStatus {
		case wire.RateLimitStatusLimited:
			code = 3
		case wire.RateLimitStatusAlert:
			code = 2
		case wire.RateLimitStatusClear:
			code = 4
		case wire.RateLimitStatusDisconnect:
			s.logger.DebugContext(ctx, "rate limit status disconnected, no point in returning status update")
			continue
		}

		msgs = append(msgs, buildRateLimitUpdate(code, curRate, instance, now))
	}

	return msgs
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

// buildRateLimitUpdate constructs a SNAC message notifying the client of a
// rate limit threshold update or a change in rate limiting status for a specific class.
//
// The message format varies depending on the client's supported protocol version.
// If OService version 2 or higher is supported,
// additional metadata such as time since last status change and whether SNACs are
// currently being dropped will be included.
func buildRateLimitUpdate(code uint16, curRate state.RateClassState, instance *state.SessionInstance, now time.Time) wire.SNACMessage {
	var droppingSNACs uint8
	if curRate.CurrentStatus == wire.RateLimitStatusLimited {
		droppingSNACs = 1
	}

	rate := wire.RateParamsSNAC{
		ID:              uint16(curRate.ID),
		WindowSize:      uint32(curRate.WindowSize),
		ClearLevel:      uint32(curRate.ClearLevel),
		AlertLevel:      uint32(curRate.AlertLevel),
		LimitLevel:      uint32(curRate.LimitLevel),
		DisconnectLevel: uint32(curRate.DisconnectLevel),
		CurrentLevel:    uint32(curRate.CurrentLevel),
		MaxLevel:        uint32(curRate.MaxLevel),
	}
	if instance.FoodGroupVersions()[wire.OService] > 1 {
		rate.V2Params = &struct {
			LastTime      uint32
			DroppingSNACs uint8
		}{
			LastTime:      uint32(max(0, now.Unix()-curRate.LastTime.Unix())),
			DroppingSNACs: droppingSNACs,
		}
	}

	return wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.OService,
			SubGroup:  wire.OServiceRateParamChange,
			RequestID: wire.ReqIDFromServer,
		},
		Body: wire.SNAC_0x01_0x0A_OServiceRateParamsChange{
			Code: code,
			Rate: rate,
		},
	}
}
