package state

import (
	"net/netip"
	"sync"
	"time"

	"github.com/pchchv/go-icq/wire"
)

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

// Session represents a user's current session.
// Unless stated otherwise,
// all methods may be safely accessed by multiple goroutines.
type Session struct {
	awayMessage             string
	buddyIcon               wire.BARTID
	caps                    [][16]byte
	chatRoomCookie          string
	clientID                string
	closed                  bool
	displayScreenName       DisplayScreenName
	foodGroupVersions       [wire.MDir + 1]uint16
	identScreenName         IdentScreenName
	idle                    bool
	idleTime                time.Time
	lastObservedStates      [5]RateClassState
	msgCh                   chan wire.SNACMessage
	multiConnFlag           wire.MultiConnFlag
	kerberosAuth            bool
	mutex                   sync.RWMutex
	nowFn                   func() time.Time
	rateLimitStates         [5]RateClassState
	rateLimitStatesOriginal [5]RateClassState
	remoteAddr              *netip.AddrPort
	signonComplete          bool
	signonTime              time.Time
	stopCh                  chan struct{}
	typingEventsEnabled     bool
	uin                     uint32
	userInfoBitmask         uint16
	userStatusBitmask       uint32
	warning                 uint16
	warningCh               chan uint16
	lastWarnUpdate          time.Time
	profile                 UserProfile
	memberSince             time.Time
	offlineMsgCount         int
}

// NewSession returns a new instance of Session.
// By default, the user may have up to 1000 pending messages before blocking.
func NewSession() *Session {
	now := time.Now()
	return &Session{
		msgCh:             make(chan wire.SNACMessage, 1000),
		nowFn:             time.Now,
		stopCh:            make(chan struct{}),
		signonTime:        now,
		caps:              make([][16]byte, 0),
		userInfoBitmask:   wire.OServiceUserFlagOSCARFree,
		userStatusBitmask: wire.OServiceUserStatusAvailable,
		foodGroupVersions: func() [wire.MDir + 1]uint16 {
			// initialize default food groups versions to 1.0
			vals := [wire.MDir + 1]uint16{}
			vals[wire.OService] = 1
			vals[wire.Locate] = 1
			vals[wire.Buddy] = 1
			vals[wire.ICBM] = 1
			vals[wire.Advert] = 1
			vals[wire.Invite] = 1
			vals[wire.Admin] = 1
			vals[wire.Popup] = 1
			vals[wire.PermitDeny] = 1
			vals[wire.UserLookup] = 1
			vals[wire.Stats] = 1
			vals[wire.Translate] = 1
			vals[wire.ChatNav] = 1
			vals[wire.Chat] = 1
			vals[wire.ODir] = 1
			vals[wire.BART] = 1
			vals[wire.Feedbag] = 1
			vals[wire.ICQ] = 1
			vals[wire.BUCP] = 1
			vals[wire.Alert] = 1
			vals[wire.Plugin] = 1
			vals[wire.UnnamedFG24] = 1
			vals[wire.MDir] = 1
			return vals
		}(),
		warningCh: make(chan uint16, 1),
	}
}

// SetRemoteAddr sets the user's remote IP address
func (s *Session) SetRemoteAddr(remoteAddr *netip.AddrPort) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.remoteAddr = remoteAddr
}

// SetAwayMessage sets the user's away message.
func (s *Session) SetAwayMessage(awayMessage string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.awayMessage = awayMessage
}

// SetChatRoomCookie sets the chatRoomCookie for the chat room the user is currently in.
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

// SetSignonComplete indicates that the client has completed the sign-on sequence.
func (s *Session) SetSignonComplete() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.signonComplete = true
}

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

// SetUserInfoFlag sets a flag to and returns UserInfoBitmask
func (s *Session) SetUserInfoFlag(flag uint16) (flags uint16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.userInfoBitmask |= flag
	return s.userInfoBitmask
}

// SetOfflineMsgCount sets the offline message count.
func (s *Session) SetOfflineMsgCount(count int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.offlineMsgCount = count
}

// SetProfile sets the user's profile information.
func (s *Session) SetProfile(profile UserProfile) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.profile = profile
}

// SetTypingEventsEnabled sets whether the client wants to send and receive
// typing events.
func (s *Session) SetTypingEventsEnabled(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.typingEventsEnabled = enabled
}

// SetKerberosAuth sets whether Kerberos authentication was used for this session.
func (s *Session) SetKerberosAuth(enabled bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.kerberosAuth = enabled
}

// SetFoodGroupVersions sets the client's supported food group versions
func (s *Session) SetFoodGroupVersions(versions [wire.MDir + 1]uint16) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.foodGroupVersions = versions
}

// SetUserStatusBitmask sets the user status bitmask from the client.
func (s *Session) SetUserStatusBitmask(bitmask uint32) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.userStatusBitmask = bitmask
}

// SetIdentScreenName sets the user's screen name.
func (s *Session) SetIdentScreenName(screenName IdentScreenName) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.identScreenName = screenName
}

// SetMultiConnFlag sets the multi-connection flag for this session.
func (s *Session) SetMultiConnFlag(flag wire.MultiConnFlag) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.multiConnFlag = flag
}

// SetSignonTime sets the user's sign-ontime.
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

// SetCaps sets capability UUIDs that represent the
// features the client supports.
// If set, capability metadata appears in the user info TLV list.
func (s *Session) SetCaps(caps [][16]byte) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.caps = caps
}

// SetDisplayScreenName sets the user's screen name.
func (s *Session) SetDisplayScreenName(displayScreenName DisplayScreenName) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.displayScreenName = displayScreenName
}

// SetIdle sets the user's idle state.
func (s *Session) SetIdle(dur time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.idle = true
	// set the time the user became idle
	s.idleTime = s.nowFn().Add(-dur)
}

// SetBuddyIcon stores the session's buddy icon metadata.
func (s *Session) SetBuddyIcon(icon wire.BARTID) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.buddyIcon = icon
}

// SetClientID sets the client ID.
func (s *Session) SetClientID(clientID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.clientID = clientID
}

// SetMemberSince sets the member since timestamp.
func (s *Session) SetMemberSince(t time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.memberSince = t
}

// RemoteAddrs returns user's remote IP address
func (s *Session) RemoteAddr() (remoteAddr *netip.AddrPort) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.remoteAddr
}

// AwayMessage returns the user's away message.
func (s *Session) AwayMessage() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.awayMessage
}

// ChatRoomCookie gets the chatRoomCookie for the chat room the user is currently in.
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

// SignonComplete indicates whether the client has completed the sign-on sequence.
func (s *Session) SignonComplete() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.signonComplete
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

// TypingEventsEnabled indicates whether the client wants to
// send and receive typing events.
func (s *Session) TypingEventsEnabled() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.typingEventsEnabled
}

// KerberosAuth indicates whether Kerberos authentication was used for this session.
func (s *Session) KerberosAuth() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.kerberosAuth
}

// FoodGroupVersions retrieves the client's supported food group versions.
func (s *Session) FoodGroupVersions() [wire.MDir + 1]uint16 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.foodGroupVersions
}

// UserStatusBitmask returns the user status bitmask.
func (s *Session) UserStatusBitmask() uint32 {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.userStatusBitmask
}
