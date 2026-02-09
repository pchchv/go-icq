package toc

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// ChatRegistry manages the chat rooms that a user is
// connected to during a TOC session.
// It maintains mappings between chat room identifiers,
// metadata, and active chat sessions.
//
// This struct provides thread-safe operations for adding, retrieving,
// and managing chat room metadata and associated sessions.
type ChatRegistry struct {
	sessions map[int]*state.SessionInstance // tracks active chat sessions by chat room ID
	lookup   map[int]wire.ICBMRoomInfo      // maps chat room IDs to their metadata
	nextID   int                            // incremental identifier for newly added chat rooms
	m        sync.RWMutex                   // synchronization primitive for concurrent access
}

// NewChatRegistry creates a new ChatRegistry instances.
func NewChatRegistry() *ChatRegistry {
	return &ChatRegistry{
		lookup:   make(map[int]wire.ICBMRoomInfo),
		sessions: make(map[int]*state.SessionInstance),
		m:        sync.RWMutex{},
	}
}

// Add registers metadata for a newly joined chat room and
// returns a unique identifier for it.
// If the room is already registered, it returns the existing ID.
func (c *ChatRegistry) Add(room wire.ICBMRoomInfo) int {
	c.m.Lock()
	defer c.m.Unlock()

	for chatID, r := range c.lookup {
		if r == room {
			return chatID
		}
	}

	id := c.nextID
	c.lookup[id] = room
	c.nextID++
	return id
}

// Sessions retrieves all the chat sessions.
func (c *ChatRegistry) Sessions() []*state.SessionInstance {
	c.m.RLock()
	defer c.m.RUnlock()

	sessions := make([]*state.SessionInstance, 0, len(c.sessions))
	for _, s := range c.sessions {
		sessions = append(sessions, s)
	}

	return sessions
}

// RegisterSess associates a chat session with a chat room.
// If a session is already registered for the given chat ID, it will be overwritten.
func (c *ChatRegistry) RegisterSess(chatID int, instance *state.SessionInstance) {
	c.m.Lock()
	defer c.m.Unlock()

	c.sessions[chatID] = instance
}

// RetrieveSess retrieves the chat session associated with the given chat ID.
// If no session is registered for the chat ID, it returns nil.
func (c *ChatRegistry) RetrieveSess(chatID int) *state.SessionInstance {
	c.m.RLock()
	defer c.m.RUnlock()

	return c.sessions[chatID]
}

// RemoveSess removes a chat session.
func (c *ChatRegistry) RemoveSess(chatID int) {
	c.m.Lock()
	defer c.m.Unlock()

	delete(c.sessions, chatID)
}

// LookupRoom retrieves metadata for the chat room registered with chatID.
// It returns the room metadata and a
// boolean indicating whether the chat ID was found.
func (c *ChatRegistry) LookupRoom(chatID int) (room wire.ICBMRoomInfo, found bool) {
	c.m.RLock()
	defer c.m.RUnlock()

	room, found = c.lookup[chatID]
	return
}

// OSCARProxy acts as a bridge between TOC clients and the OSCAR server,
// translating protocol messages between the two.
//
// It performs the following functions:
//   - Receives TOC messages from the client, converts them into SNAC messages,
//     and forwards them to the OSCAR server.
//     The SNAC response is then converted back into a TOC response for the client.
//   - Receives incoming messages from the
//     OSCAR server and translates them into TOC responses for the client.
type OSCARProxy struct {
	AdminService      AdminService
	AuthService       AuthService
	BuddyListRegistry BuddyListRegistry
	BuddyService      BuddyService
	ChatNavService    ChatNavService
	ChatService       ChatService
	CookieBaker       CookieBaker
	DirSearchService  DirSearchService
	ICBMService       ICBMService
	LocateService     LocateService
	Logger            *slog.Logger
	OServiceService   OServiceService
	PermitDenyService PermitDenyService
	TOCConfigStore    TOCConfigStore
	SessionRetriever  SessionRetriever
	SNACRateLimits    wire.SNACRateLimits
	HTTPIPRateLimiter *IPRateLimiter
}

// AddBuddy handles the toc_add_buddy TOC command.
//
// From the TiK documentation: Add buddies to your buddy list. This does not change your saved config.
// Command syntax: toc_add_buddy <Buddy User 1> [<Buddy User2> [<Buddy User 3> [...]]]
func (s OSCARProxy) AddBuddy(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if msg, isLimited := s.checkRateLimit(ctx, me, wire.Buddy, wire.BuddyAddBuddies); isLimited {
		return msg
	}

	users, err := parseArgs(args)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	snac := wire.SNAC_0x03_0x04_BuddyAddBuddies{}
	for _, sn := range users {
		snac.Buddies = append(snac.Buddies, struct {
			ScreenName string `oscar:"len_prefix=uint8"`
		}{ScreenName: sn})
	}

	if err := s.BuddyService.AddBuddies(ctx, me, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("BuddyService.AddBuddies: %w", err))
	}

	return ""
}

// AddPermit handles the toc_add_permit TOC command.
//
// From the TiK documentation:
//
//	ADD the following people to your permit mode.
//	If you are in deny mode it	will switch you to permit mode first.
//	With no arguments and in deny mode	this will switch you to permit none.
//	If already in permit mode,
//	no arguments does nothing and your permit list remains the same.
//
// Command syntax: toc_add_permit [ <User 1> [<User 2> [...]]]
func (s OSCARProxy) AddPermit(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if msg, isLimited := s.checkRateLimit(ctx, me, wire.PermitDeny, wire.PermitDenyAddDenyListEntries); isLimited {
		return msg
	}

	users, err := parseArgs(args)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	snac := wire.SNAC_0x09_0x05_PermitDenyAddPermListEntries{}
	for _, sn := range users {
		snac.Users = append(snac.Users, struct {
			ScreenName string `oscar:"len_prefix=uint8"`
		}{ScreenName: sn})
	}

	if err := s.PermitDenyService.AddPermListEntries(ctx, me, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("PermitDenyService.AddPermListEntries: %w", err))
	}

	return ""
}

// AddDeny handles the toc_add_deny TOC command.
//
// From the TiK documentation:
//
//	ADD the following people to your deny mode.
//	If you are in permit mode it will switch you to deny mode first.
//	With no arguments and in permit mode, this will switch you to deny none.
//	If already in deny mode, no arguments does nothing and your deny list remains unchanged.
//
// Command syntax: toc_add_deny [ <User 1> [<User 2> [...]]]
func (s OSCARProxy) AddDeny(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if msg, isLimited := s.checkRateLimit(ctx, me, wire.PermitDeny, wire.PermitDenyAddDenyListEntries); isLimited {
		return msg
	}

	users, err := parseArgs(args)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	snac := wire.SNAC_0x09_0x07_PermitDenyAddDenyListEntries{}
	for _, sn := range users {
		snac.Users = append(snac.Users, struct {
			ScreenName string `oscar:"len_prefix=uint8"`
		}{ScreenName: sn})
	}

	if err := s.PermitDenyService.AddDenyListEntries(ctx, me, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("PermitDenyService.AddDenyListEntries: %w", err))
	}

	return ""
}

func (s OSCARProxy) checkRateLimit(ctx context.Context, sender *state.SessionInstance, foodGroup uint16, subGroup uint16) (string, bool) {
	rateClassID, ok := s.SNACRateLimits.RateClassLookup(foodGroup, subGroup)
	if !ok {
		s.Logger.ErrorContext(ctx, "rate limit not found, allowing request through")
		return "", false
	}

	if status := sender.Session().EvaluateRateLimit(time.Now(), rateClassID); status != wire.RateLimitStatusLimited {
		return "", false
	}

	s.Logger.DebugContext(
		ctx,
		"(toc) rate limit exceeded, dropping SNAC",
		"foodgroup",
		wire.FoodGroupName(foodGroup),
		"subgroup",
		wire.SubGroupName(foodGroup, subGroup),
	)
	return rateLimitExceededErr, true
}

// runtimeErr is a convenience function that logs an
// error and returns a TOC internal server error.
func (s OSCARProxy) runtimeErr(ctx context.Context, err error) string {
	s.Logger.ErrorContext(ctx, "internal service error", "err", err.Error())
	return cmdInternalSvcErr
}

// parseArgs extracts arguments from a TOC command.
// Each positional argument is assigned to its corresponding args pointer.
// It returns the remaining arguments as varargs.
func parseArgs(payload []byte, args ...*string) (varArgs []string, err error) {
	if len(payload) == 0 && len(args) == 0 {
		return []string{}, nil
	}

	reader := csv.NewReader(bytes.NewReader(payload))
	reader.Comma = ' '
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	segs, err := reader.Read()
	if err != nil {
		return []string{}, fmt.Errorf("CSV reader error: %w", err)
	} else if len(segs) < len(args) {
		return []string{}, fmt.Errorf("command contains fewer arguments than expected")
	}

	// populate placeholder pointers with their corresponding values
	for i, arg := range args {
		if arg != nil {
			*arg = strings.TrimSpace(segs[i])
		}
	}

	// dump remaining arguments as varargs
	return segs[len(args):], err
}
