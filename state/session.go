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
