package state

import (
	"net/netip"
	"sync"
	"time"

	"github.com/pchchv/go-icq/wire"
)

const (
	// SessSendOK indicates message was sent to recipient.
	SessSendOK SessSendStatus = iota
	// SessQueueFull indicates send failed due to full queue -- client is likely dead.
	SessQueueFull
	// SessSendClosed indicates send did not complete because session is closed.
	SessSendClosed
)

// SessSendStatus is the result of sending a message to a user.
type SessSendStatus int

// RateClassState tracks the rate limiting state for a
// specific rate class within a user's session.
//
// It embeds the static wire.RateClass configuration and maintains dynamic,
// per-session state used to evaluate rate limits in real time.
type RateClassState struct {
	// static rate limit configuration for this class
	wire.RateClass
	// CurrentLevel is the current exponential moving average for this rate class.
	CurrentLevel int32
	// LastTime represents the last time a SNAC message was sent for this rate class.
	LastTime time.Time
	// CurrentStatus is the last recorded rate limit status for this rate class.
	CurrentStatus wire.RateLimitStatus
	// Subscribed indicates whether the user wants to
	// receive rate limit parameter updates for this rate class.
	Subscribed bool
	// LimitedNow indicates whether the user is currently rate limited for this rate class.
	// The user is blocked from sending SNACs in this rate class until the clear threshold is met.
	LimitedNow bool
}

// Session represents shared user-level state that persists across all concurrent
// connections for a single user account.
//
// Session maintains client identity information, preferences, rate limiting state,
// and other shared data that should be consistent across all of a user's active
// connections. Individual connection-specific state (like remote address, sign-on
// status, or per-connection capabilities) is stored in SessionInstance instead.
//
// All methods on Session are safe for concurrent use.
type Session struct {
	mutex sync.RWMutex
	// Rate limiting (shared across all sessions per user)
	nowFn                   func() time.Time
	rateLimitStates         [5]RateClassState
	rateLimitStatesOriginal [5]RateClassState
	lastObservedStates      [5]RateClassState
	instancesOrdered        []*SessionInstance
	onSessCloseFn           func()
	instances               map[uint8]*SessionInstance
	initOnce                sync.Once
	// User-level settings and profile (shared)
	warning             uint16
	warningCh           chan uint16
	buddyIcon           wire.BARTID
	chatRoomCookie      string
	offlineMsgCount     int
	typingEventsEnabled bool
	// User identity (shared across all sessions)
	displayScreenName DisplayScreenName
	identScreenName   IdentScreenName
	memberSince       time.Time
	signonTime        time.Time
	uin               uint32
}

// NewSession returns a new instance of Session.
func NewSession() *Session {
	return &Session{
		warningCh:        make(chan uint16, 1),
		instances:        make(map[uint8]*SessionInstance),
		instancesOrdered: make([]*SessionInstance, 0),
		onSessCloseFn:    func() {},
		nowFn:            time.Now,
	}
}

// SetChatRoomCookie sets the chat room cookie.
func (s *Session) SetChatRoomCookie(cookie string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.chatRoomCookie = cookie
}

// SetUIN sets the user's ICQ number.
func (s *Session) SetUIN(uin uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.uin = uin
}

// SetRateClasses sets the rate limit classes (shared across all sessions).
func (s *Session) SetRateClasses(now time.Time, classes wire.RateLimitClasses) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var newStates [5]RateClassState
	for i, class := range classes.All() {
		newStates[i] = RateClassState{
			CurrentLevel:  class.MaxLevel,
			CurrentStatus: wire.RateLimitStatusClear,
			LastTime:      now,
			RateClass:     class,
			Subscribed:    s.lastObservedStates[i].Subscribed,
		}
	}

	if s.lastObservedStates[0].ID == 0 {
		s.lastObservedStates = newStates
	} else {
		s.lastObservedStates = s.rateLimitStates
	}

	s.rateLimitStates = newStates
	s.rateLimitStatesOriginal = newStates
}

// SetOfflineMsgCount sets the offline message count.
func (s *Session) SetOfflineMsgCount(count int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.offlineMsgCount = count
}

// SetTypingEventsEnabled sets whether the session wants to send and receive typing events.
func (s *Session) SetTypingEventsEnabled(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.typingEventsEnabled = enabled
}

// SetIdentScreenName sets the user's identity screen name (shared across all sessions).
func (s *Session) SetIdentScreenName(screenName IdentScreenName) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.identScreenName = screenName
}

// SetSignonTime sets the session's sign-on time.
func (s *Session) SetSignonTime(t time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.signonTime = t
}

// SetWarning sets the user's last warning level.
func (s *Session) SetWarning(warning uint16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.warning = warning
}

// SetDisplayScreenName sets the user's screen name.
func (s *Session) SetDisplayScreenName(displayScreenName DisplayScreenName) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.displayScreenName = displayScreenName
}

// SetBuddyIcon stores the session's buddy icon metadata.
func (s *Session) SetBuddyIcon(icon wire.BARTID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.buddyIcon = icon
}

// SetMemberSince sets the member since timestamp.
func (s *Session) SetMemberSince(t time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.memberSince = t
}

// AwayMessage returns the user's away message.
func (s *Session) AwayMessage() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.awayMessage
}

// ChatRoomCookie returns the chat room cookie.
func (s *Session) ChatRoomCookie() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.chatRoomCookie
}

// UIN returns the user's ICQ number.
func (s *Session) UIN() uint32 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.uin
}

// OfflineMsgCount returns the offline message count.
func (s *Session) OfflineMsgCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.offlineMsgCount
}

// Profile returns the user's profile information.
func (s *Session) Profile() UserProfile {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.profile
}

// TypingEventsEnabled indicates whether the session wants to
// send and receive typing events.
func (s *Session) TypingEventsEnabled() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.typingEventsEnabled
}

// IdentScreenName returns the user's identity screen name.
func (s *Session) IdentScreenName() IdentScreenName {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.identScreenName
}

// SignonTime returns the session's sign-on time.
func (s *Session) SignonTime() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.signonTime
}

// Warning returns the user's current warning level.
func (s *Session) Warning() uint16 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.warning
}

// WarningCh returns the warning notification channel.
func (s *Session) WarningCh() chan uint16 {
	return s.warningCh
}

// Caps retrieves user capabilities.
func (s *Session) Caps() [][16]byte {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.caps
}

// DisplayScreenName returns the user's display screen name.
func (s *Session) DisplayScreenName() DisplayScreenName {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.displayScreenName
}

// Idle reports the user's idle state.
func (s *Session) Idle() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.idle
}

// IdleTime reports when the user went idle
func (s *Session) IdleTime() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.idleTime
}


// BuddyIcon returns the session's buddy icon metadata and
// reports whether it has been set.
func (s *Session) BuddyIcon() (wire.BARTID, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	icon := s.buddyIcon
	return icon, icon.Type != 0
}

// MemberSince reports when the user became a member.
func (s *Session) MemberSince() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.memberSince
}

// ScaleWarningAndRateLimit increments the user's warning level and scales rate limits.
func (s *Session) ScaleWarningAndRateLimit(incr int16, classID wire.RateLimitClassID) (bool, uint16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// handle warning level increment
	newWarning := int32(s.warning) + int32(incr)
	if newWarning < 0 {
		s.warning = 0 // clamp min at 0
	} else if newWarning > 1000 {
		return false, 0
	} else {
		s.warning = uint16(newWarning)
	}

	pct := float32(incr) / 1000.0
	// create reference variables for better readability
	rateClass := &s.rateLimitStates[classID-1]
	originalRateClass := &s.rateLimitStatesOriginal[classID-1]
	// clamp function to constrain values between min and max
	clamp := func(value, min, max int32) int32 {
		if value < min {
			return min
		} else if value > max {
			return max
		} else {
			return value
		}
	}

	// apply a buffer to limit/clear/alert levels so that
	// they never approach too close to the maximum level
	// otherwise, AIM 4.8 exhibits instability
	// (client crashes, IM window glitches)
	// when the warning level reaches 90-100%
	maxLevel := originalRateClass.MaxLevel - 150
	// scale the rate limit parameters
	newLimitLevel := rateClass.LimitLevel + int32(float32(maxLevel-originalRateClass.LimitLevel)*pct)
	rateClass.LimitLevel = clamp(newLimitLevel, originalRateClass.LimitLevel, originalRateClass.MaxLevel)

	newLimitLevel = rateClass.ClearLevel + int32(float32(maxLevel-originalRateClass.ClearLevel)*pct)
	rateClass.ClearLevel = clamp(newLimitLevel, originalRateClass.ClearLevel, originalRateClass.MaxLevel)

	newLimitLevel = rateClass.AlertLevel + int32(float32(maxLevel-originalRateClass.AlertLevel)*pct)
	rateClass.AlertLevel = clamp(newLimitLevel, originalRateClass.AlertLevel, originalRateClass.MaxLevel)

	s.warningCh <- s.warning
	return true, s.warning
}

// RateLimitStates returns the current rate limit states (shared across all sessions).
func (s *Session) RateLimitStates() [5]RateClassState {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.rateLimitStates
}

// SubscribeRateLimits subscribes to rate limit updates.
func (s *Session) SubscribeRateLimits(classes []wire.RateLimitClassID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, classID := range classes {
		s.rateLimitStates[classID-1].Subscribed = true
	}
}

// EvaluateRateLimit checks and updates the
// session’s rate limit state for the given rate class ID.
// If the rate status reaches 'disconnect', the session is closed.
// Rate limits are not enforced if the user is a bot
// (has wire.OServiceUserFlagBot set in their user info bitmask).
func (s *Session) EvaluateRateLimit(now time.Time, rateClassID wire.RateLimitClassID) wire.RateLimitStatus {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.userInfoBitmask&wire.OServiceUserFlagBot == wire.OServiceUserFlagBot {
		return wire.RateLimitStatusClear // don't rate limit bots
	}

	rateClass := &s.rateLimitStates[rateClassID-1]
	status, newLevel := wire.CheckRateLimit(rateClass.LastTime, now, rateClass.RateClass, rateClass.CurrentLevel, rateClass.LimitedNow)
	rateClass.CurrentLevel = newLevel
	rateClass.CurrentStatus = status
	rateClass.LastTime = now
	rateClass.LimitedNow = status == wire.RateLimitStatusLimited
	if status == wire.RateLimitStatusDisconnect {
		s.close()
	}

	return status
}

// ObserveRateChanges updates rate limit states and returns changes.
func (s *Session) ObserveRateChanges(now time.Time) (classDelta []RateClassState, stateDelta []RateClassState) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for i, params := range s.rateLimitStates {
		if !params.Subscribed {
			continue
		}

		state, level := wire.CheckRateLimit(params.LastTime, now, params.RateClass, params.CurrentLevel, params.LimitedNow)
		s.rateLimitStates[i].CurrentStatus = state
		// clear limited now flag if passing from limited state to clear state
		if s.rateLimitStates[i].LimitedNow && state == wire.RateLimitStatusClear {
			s.rateLimitStates[i].LimitedNow = false
			s.rateLimitStates[i].CurrentLevel = level
		}

		// did rate class change?
		if params.RateClass != s.lastObservedStates[i].RateClass {
			classDelta = append(classDelta, s.rateLimitStates[i])
		}

		// did rate limit status change?
		if s.lastObservedStates[i].CurrentStatus != s.rateLimitStates[i].CurrentStatus {
			stateDelta = append(stateDelta, s.rateLimitStates[i])
		}

		// save it for next time
		s.lastObservedStates[i] = s.rateLimitStates[i]
	}

	return classDelta, stateDelta
}

// Close shuts down the session's ability to relay messages.
// Once invoked, RelayMessage returns SessQueueFull and Closed returns a closed channel.
// It is not possible to re-open message relaying once closed.
// It is safe to call from multiple go routines.
func (s *Session) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.close()
}

// RelayMessage receives a SNAC message from a user and passes it on
// asynchronously to the consumer of this session's messages.
// It returns SessSendStatus to indicate whether the message was successfully sent or not.
// This method is non-blocking.
func (s *Session) RelayMessage(msg wire.SNACMessage) SessSendStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.closed {
		return SessSendClosed
	}

	select {
	case s.msgCh <- msg:
		return SessSendOK
	case <-s.stopCh:
		return SessSendClosed
	default:
		return SessQueueFull
	}
}

// TLVUserInfo returns a TLV list containing session information aggregated from all instances.
func (s *Session) TLVUserInfo() wire.TLVUserInfo {
	return wire.TLVUserInfo{
		ScreenName:   s.DisplayScreenName().String(),
		WarningLevel: s.Warning(),
		TLVBlock: wire.TLVBlock{
			TLVList: s.userInfo(),
		},
	}
}

// Invisible returns true if the user is idle.
func (s *Session) Invisible() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.userStatusBitmask&wire.OServiceUserStatusInvisible == wire.OServiceUserStatusInvisible
}

func (s *Session) close() {
	if s.closed {
		return
	}

	close(s.stopCh)
	s.closed = true
}

func (s *Session) userInfo() wire.TLVList {
	tlvs := wire.TLVList{}

	// sign-in timestamp
	tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoSignonTOD, uint32(s.signonTime.Unix())))

	// user info flags
	uFlags := s.userInfoBitmask
	if s.awayMessage != "" {
		uFlags |= wire.OServiceUserFlagUnavailable
	}
	tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoUserFlags, uFlags))

	// user status flags
	tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoStatus, s.userStatusBitmask))

	// idle status
	if s.idle {
		tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoIdleTime, uint16(s.nowFn().Sub(s.idleTime).Minutes())))
	}

	// set buddy icon metadata, if user has buddy icon
	if bartID, hasIcon := s.BuddyIcon(); hasIcon {
		tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoBARTInfo, bartID))
	}

	// ICQ direct-connect info. The TLV is required for buddy arrival events to
	// work in ICQ, even if the values are set to default.
	if s.userInfoBitmask&wire.OServiceUserFlagICQ == wire.OServiceUserFlagICQ {
		tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoICQDC, wire.ICQDCInfo{}))
	}

	// capabilities (buddy icon, chat, etc...)
	if len(s.caps) > 0 {
		tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoOscarCaps, s.caps))
	}

	tlvs.Append(wire.NewTLVBE(wire.OServiceUserInfoMySubscriptions, uint32(0)))
	return tlvs
}

// SessionInstance represents a single client connection instance
// within a user's session.
// Multiple SessionInstance objects can belong to the same Session,
// allowing a user to maintain concurrent connections from
// different clients or devices.
//
// SessionInstance stores connection-specific state such as the remote address,
// sign-on completion status, client capabilities, idle state,
// and per-connection profile data.
// It holds a reference to its parent Session to access shared
// user-level data like identity, warning levels, and rate limiting state.
//
// All methods on SessionInstance are safe for concurrent use.
type SessionInstance struct {
	session *Session
	mutex   sync.RWMutex
	// Unique instance identifier
	instanceNum uint8
	// Per-session connection state
	remoteAddr     *netip.AddrPort
	signonComplete bool
	closed         bool
	stopCh         chan struct{}
	msgCh          chan wire.SNACMessage
	kerberosAuth   bool
	// Per-session client information
	clientID          string
	capabilities      [][16]byte
	foodGroupVersions [wire.MDir + 1]uint16
	multiConnFlag     wire.MultiConnFlag
	// Per-session state
	idle              bool
	idleTime          time.Time
	awayMsg           string
	userInfoBitmask   uint16
	userStatusBitmask uint32
	// Per-session profile
	profile           UserProfile
	awayTime          time.Time
	onInstanceCloseFn func()
}

// Session returns the parent Session for this instance.
func (s *SessionInstance) Session() *Session {
	return s.session
}

// ClientID retrieves the instance's client ID.
func (s *SessionInstance) ClientID() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.clientID
}

// ChatRoomCookie returns the chat room cookie from the parent session.
func (s *SessionInstance) ChatRoomCookie() string {
	return s.session.ChatRoomCookie()
}

// UIN returns the user's ICQ number.
func (s *SessionInstance) UIN() uint32 {
	return s.session.UIN()
}

// Num returns the unique instance identifier.
func (s *SessionInstance) Num() uint8 {
	return s.instanceNum
}

// DisplayScreenName returns the user's display screen name.
func (s *SessionInstance) DisplayScreenName() DisplayScreenName {
	return s.session.DisplayScreenName()
}

// IdentScreenName returns the user's identity screen name.
func (s *SessionInstance) IdentScreenName() IdentScreenName {
	return s.session.IdentScreenName()
}

// Away returns true if the instance is away.
func (s *SessionInstance) Away() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.away()
}

// AwayMessage returns the instance's away message and the time it was set.
func (s *SessionInstance) AwayMessage() (string, time.Time) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.awayMsg, s.awayTime
}

// Idle reports the instance's idle state.
func (s *SessionInstance) Idle() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.idle
}

// IdleTime reports when the instance went idle.
func (s *SessionInstance) IdleTime() time.Time {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.idleTime
}

// Invisible returns true if the user is invisible.
func (s *SessionInstance) Invisible() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.userStatusBitmask&wire.OServiceUserStatusInvisible == wire.OServiceUserStatusInvisible
}

// SignonComplete indicates whether the instance has completed the sign-on sequence.
func (s *SessionInstance) SignonComplete() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.signonComplete
}

// RateLimitStates returns the current rate limit states.
func (s *SessionInstance) RateLimitStates() [5]RateClassState {
	return s.session.RateLimitStates()
}

// Warning returns the user's current warning level.
func (s *SessionInstance) Warning() uint16 {
	return s.session.Warning()
}

// WarningCh returns the warning notification channel.
func (s *SessionInstance) WarningCh() chan uint16 {
	return s.session.WarningCh()
}

// Closed blocks until the instance is closed.
func (s *SessionInstance) Closed() <-chan struct{} {
	return s.stopCh
}

// OnClose registers a function to be called when the instance closes,
// but only if other instances remain in the session.
// If this is the last instance to close,
// OnSessionClose will be called instead.
func (s *SessionInstance) OnClose(fn func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.onInstanceCloseFn = fn
}

// ClearUserInfoFlag clears a flag from the user info bitmask.
func (s *SessionInstance) ClearUserInfoFlag(flag uint16) (flags uint16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.userInfoBitmask &^= flag
	return s.userInfoBitmask
}

// KerberosAuth indicates whether Kerberos authentication was used for this instance.
func (s *SessionInstance) KerberosAuth() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.kerberosAuth
}

// Profile returns the user's profile information.
func (s *SessionInstance) Profile() UserProfile {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.profile
}

// FoodGroupVersions retrieves the instance's supported food group versions.
func (s *SessionInstance) FoodGroupVersions() [wire.MDir + 1]uint16 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.foodGroupVersions
}

// RemoteAddr returns the instance's remote IP address.
func (s *SessionInstance) RemoteAddr() (remoteAddr *netip.AddrPort) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.remoteAddr
}

// MultiConnFlag retrieves the multi-connection flag for this instance.
func (s *SessionInstance) MultiConnFlag() wire.MultiConnFlag {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.multiConnFlag
}

// UserInfoBitmask returns the user info bitmask.
func (s *SessionInstance) UserInfoBitmask() uint16 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.userInfoBitmask
}

// UserStatusBitmask returns the user status bitmask.
func (s *SessionInstance) UserStatusBitmask() uint32 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.userStatusBitmask
}

// SignonTime returns the session's sign-on time.
func (s *SessionInstance) SignonTime() time.Time {
	return s.session.SignonTime()
}

// TypingEventsEnabled indicates whether the session wants to send and receive typing events.
func (s *SessionInstance) TypingEventsEnabled() bool {
	return s.session.TypingEventsEnabled()
}

// ReceiveMessage returns a channel of messages relayed via this instance.
func (s *SessionInstance) ReceiveMessage() chan wire.SNACMessage {
	return s.msgCh
}

// RelayMessageToInstance receives a SNAC message and passes it to the instance's message channel.
func (s *SessionInstance) RelayMessageToInstance(msg wire.SNACMessage) SessSendStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if s.closed {
		return SessSendClosed
	}

	select {
	case s.msgCh <- msg:
		return SessSendOK
	case <-s.stopCh:
		return SessSendClosed
	default:
		return SessQueueFull
	}
}

// OfflineMsgCount returns the offline message count.
func (s *SessionInstance) OfflineMsgCount() int {
	return s.session.OfflineMsgCount()
}

// SetUserStatusBitmask sets the user status bitmask.
func (s *SessionInstance) SetUserStatusBitmask(bitmask uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if (bitmask&wire.OServiceUserStatusAway == wire.OServiceUserStatusAway) && !s.away() {
		s.awayTime = s.session.nowFn()
	}

	s.userStatusBitmask = bitmask
}

// SetAwayMessage sets the instance's away message.
func (s *SessionInstance) SetAwayMessage(awayMessage string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.awayMsg = awayMessage
}

// SetCaps sets capability UUIDs for the instance.
func (s *SessionInstance) SetCaps(caps [][16]byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.capabilities = caps
}

// SetRemoteAddr sets the instance's remote IP address.
func (s *SessionInstance) SetRemoteAddr(remoteAddr *netip.AddrPort) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.remoteAddr = remoteAddr
}

// SetUserInfoFlag sets a flag in the user info bitmask.
func (s *SessionInstance) SetUserInfoFlag(flag uint16) (flags uint16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if flag == wire.OServiceUserFlagUnavailable {
		s.awayTime = s.session.nowFn()
	}

	s.userInfoBitmask |= flag
	return s.userInfoBitmask
}

// SetIdle sets the instance's idle state.
func (s *SessionInstance) SetIdle(dur time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.idle = true
	// set the time the instance became idle
	s.idleTime = s.session.nowFn().Add(-dur)
}

// SetClientID sets the instance's client ID.
func (s *SessionInstance) SetClientID(clientID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.clientID = clientID
}

// SetProfile sets the user's profile information.
func (s *SessionInstance) SetProfile(profile UserProfile) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.profile = profile
}

// SetMultiConnFlag sets the multi-connection flag for this instance.
func (s *SessionInstance) SetMultiConnFlag(flag wire.MultiConnFlag) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.multiConnFlag = flag
}

// SetKerberosAuth sets whether Kerberos authentication was used for this instance.
func (s *SessionInstance) SetKerberosAuth(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.kerberosAuth = enabled
}

// SetFoodGroupVersions sets the instance's supported food group versions.
func (s *SessionInstance) SetFoodGroupVersions(versions [wire.MDir + 1]uint16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.foodGroupVersions = versions
}

// SetSignonComplete indicates that the instance has completed the sign-on sequence.
func (s *SessionInstance) SetSignonComplete() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.signonComplete = true
}

// UnsetIdle removes the instance's idle state.
func (s *SessionInstance) UnsetIdle() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.idle = false
}

// away checks if the instance is away based on bitmask flags.
// This method must be called while holding the mutex lock.
func (s *SessionInstance) away() bool {
	return s.userInfoBitmask&wire.OServiceUserFlagUnavailable != 0 ||
		s.userStatusBitmask&wire.OServiceUserStatusAway != 0
}

// caps retrieves instance capabilities.
func (s *SessionInstance) caps() [][16]byte {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.capabilities
}

// active returns true if the instance is active.
// An instance is considered active if:
// - it is not closed
// - it has completed the sign-on sequence
// - it is not idle
// - it is not away
func (s *SessionInstance) active() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return !s.closed && s.signonComplete && !s.idle && !s.away()
}

// live returns whether the instance is ready to receive messages.
func (s *SessionInstance) live() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return !s.closed && s.signonComplete
}
