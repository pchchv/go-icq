package toc

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/config"
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

// ChatJoin handles the toc_chat_join TOC command.
//
// From the TiK documentation:
//
//	Join a chat room in the given exchange.
//	Exchange is an integer that represents a group of chat rooms.
//	Different exchanges have different properties.
//	For example some exchanges might have room replication
//	(i. e. a room never fills up, there are just multiple instances)
//	and some exchanges might have navigational information.
//	Currently, exchange should always be 4, however this may change in the future.
//	You will either receive an ERROR if the room couldn't be joined or a CHAT_JOIN message.
//	The Chat Room Name is case-insensitive and consecutive spaces are removed.
//
// Command syntax: toc_chat_join <Exchange> <Chat Room Name>
func (s OSCARProxy) ChatJoin(ctx context.Context, me *state.SessionInstance, chatRegistry *ChatRegistry, args []byte) (int, string) {
	var exchangeStr, roomName string
	if _, err := parseArgs(args, &exchangeStr, &roomName); err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	// create room or retrieve the room if it already exists
	roomName = unescape(roomName)
	exchange, err := strconv.Atoi(exchangeStr)
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("strconv.Atoi: %w", err))
	}

	if msg, isLimited := s.checkRateLimit(ctx, me, wire.Chat, wire.ChatRoomInfoUpdate); isLimited {
		return 0, msg
	}

	mkRoomReq := wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate{
		Exchange: uint16(exchange),
		Cookie:   "create",
		TLVBlock: wire.TLVBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.ChatRoomTLVRoomName, roomName),
			},
		},
	}
	mkRoomReply, err := s.ChatNavService.CreateRoom(ctx, me, wire.SNACFrame{}, mkRoomReq)
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("ChatNavService.CreateRoom: %w", err))
	}

	mkRoomReplyBody, ok := mkRoomReply.Body.(wire.SNAC_0x0D_0x09_ChatNavNavInfo)
	if !ok {
		return 0, s.runtimeErr(
			ctx,
			fmt.Errorf("chatNavService.CreateRoom: unexpected response type %v", mkRoomReplyBody),
		)
	}

	buf, ok := mkRoomReplyBody.Bytes(wire.ChatNavTLVRoomInfo)
	if !ok {
		return 0, s.runtimeErr(ctx, errors.New("mkRoomReplyBody.Bytes: missing wire.ChatNavTLVRoomInfo"))
	}

	inBody := wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate{}
	if err := wire.UnmarshalBE(&inBody, bytes.NewReader(buf)); err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("wire.UnmarshalBE: %w", err))
	}

	if msg, isLimited := s.checkRateLimit(ctx, me, wire.OService, wire.OServiceServiceRequest); isLimited {
		return 0, msg
	}

	svcReqSNAC := wire.SNAC_0x01_0x04_OServiceServiceRequest{
		FoodGroup: wire.Chat,
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(0x01, wire.SNAC_0x01_0x04_TLVRoomInfo{
					Cookie: inBody.Cookie,
				}),
			},
		},
	}
	svcReqReply, err := s.OServiceService.ServiceRequest(ctx, wire.BOS, me, wire.SNACFrame{}, svcReqSNAC, config.Listener{})
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("OServiceServiceBOS.ServiceRequest: %w", err))
	}

	svcReqReplyBody, ok := svcReqReply.Body.(wire.SNAC_0x01_0x05_OServiceServiceResponse)
	if !ok {
		return 0, s.runtimeErr(ctx, fmt.Errorf("OServiceServiceBOS.ServiceRequest: unexpected response type %v", svcReqReplyBody))
	}

	loginCookie, hasCookie := svcReqReplyBody.Bytes(wire.OServiceTLVTagsLoginCookie)
	if !hasCookie {
		return 0, s.runtimeErr(ctx, errors.New("svcReqReplyBody.Bytes: missing wire.OServiceTLVTagsLoginCookie"))
	}

	// TODO: naming for cookie: login cookie, server cookie, or auth cookie?
	serverCookie, err := s.AuthService.CrackCookie(loginCookie)
	chatSess, err := s.AuthService.RegisterChatSession(ctx, serverCookie)
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("AuthService.RegisterChatSession: %w", err))
	}

	if msg, isLimited := s.checkRateLimit(ctx, me, wire.OService, wire.OServiceClientOnline); isLimited {
		return 0, msg
	}

	if err := s.OServiceService.ClientOnline(ctx, wire.Chat, wire.SNAC_0x01_0x02_OServiceClientOnline{}, chatSess); err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("OServiceServiceChat.ClientOnline: %w", err))
	}

	roomInfo := wire.ICBMRoomInfo{
		Exchange: inBody.Exchange,
		Cookie:   inBody.Cookie,
		Instance: inBody.InstanceNumber,
	}
	chatID := chatRegistry.Add(roomInfo)
	chatRegistry.RegisterSess(chatID, chatSess)
	return chatID, fmt.Sprintf("CHAT_JOIN:%d:%s", chatID, roomName)
}

// ChatLeave handles the toc_chat_leave TOC command.
//
// From the TiK documentation: leave the chat room.
//
// Command syntax: toc_chat_leave <Chat Room ID>
func (s OSCARProxy) ChatLeave(ctx context.Context, chatRegistry *ChatRegistry, args []byte) string {
	var chatIDStr string
	if _, err := parseArgs(args, &chatIDStr); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("strconv.Atoi: %w", err))
	}

	me := chatRegistry.RetrieveSess(chatID)
	if me == nil {
		return s.runtimeErr(ctx, fmt.Errorf("chatRegistry.RetrieveSess: chat session `%d` not found", chatID))
	}

	s.AuthService.SignoutChat(ctx, me)
	me.CloseInstance() // stop async server SNAC reply handler for this chat room
	chatRegistry.RemoveSess(chatID)
	return fmt.Sprintf("CHAT_LEFT:%d", chatID)
}

// ChatInvite handles the toc_chat_invite TOC command.
//
// From the TiK documentation:
//
//	Once you are inside a chat room you can invite other people into that room.
//	Remember to quote and encode the invite message.
//
// Command syntax: toc_chat_invite <Chat Room ID> <Invite Msg> <buddy1> [<buddy2> [<buddy3> [...]]]
func (s OSCARProxy) ChatInvite(ctx context.Context, me *state.SessionInstance, chatRegistry *ChatRegistry, args []byte) string {
	var chatRoomIDStr, msg string
	users, err := parseArgs(args, &chatRoomIDStr, &msg)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	msg = unescape(msg)
	chatID, err := strconv.Atoi(chatRoomIDStr)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("strconv.Atoi: %w", err))
	}

	roomInfo, found := chatRegistry.LookupRoom(chatID)
	if !found {
		return s.runtimeErr(ctx, fmt.Errorf("chatRegistry.LookupRoom: chat ID `%d` not found", chatID))
	}

	for _, guest := range users {
		if msg, isLimited := s.checkRateLimit(ctx, me, wire.ICBM, wire.ICBMChannelMsgToHost); isLimited {
			return msg
		}

		snac := wire.SNAC_0x04_0x06_ICBMChannelMsgToHost{
			ChannelID:  wire.ICBMChannelRendezvous,
			ScreenName: guest,
			TLVRestBlock: wire.TLVRestBlock{
				TLVList: wire.TLVList{
					wire.NewTLVBE(wire.ICBMTLVData, wire.ICBMCh2Fragment{
						Type:       wire.ICBMRdvMessagePropose,
						Capability: wire.CapChat,
						TLVRestBlock: wire.TLVRestBlock{
							TLVList: wire.TLVList{
								wire.NewTLVBE(wire.ICBMRdvTLVTagsSeqNum, uint16(1)),
								wire.NewTLVBE(wire.ICBMRdvTLVTagsInvitation, msg),
								wire.NewTLVBE(wire.ICBMRdvTLVTagsInviteMIMECharset, "us-ascii"),
								wire.NewTLVBE(wire.ICBMRdvTLVTagsInviteMIMELang, "en"),
								wire.NewTLVBE(wire.ICBMRdvTLVTagsSvcData, roomInfo),
							},
						},
					}),
				},
			},
		}

		if _, err := s.ICBMService.ChannelMsgToHost(ctx, me, wire.SNACFrame{}, snac); err != nil {
			return s.runtimeErr(ctx, fmt.Errorf("ICBMService.ChannelMsgToHost: %w", err))
		}
	}

	return ""
}

// ChatSend handles the toc_chat_send TOC command.
//
// From the TiK documentation:
//
//	Send a message in a chat room using the chat room id from CHAT_JOIN.
//	Since reflection is always on in TOC,
//	you do not need to add the message to your chat UI,
//	since you will get a CHAT_IN with the message.
//	Remember to quote and encode the message.
//
// Command syntax: toc_chat_send <Chat Room ID> <Message>
func (s OSCARProxy) ChatSend(ctx context.Context, chatRegistry *ChatRegistry, args []byte) string {
	var chatIDStr, msg string
	if _, err := parseArgs(args, &chatIDStr, &msg); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	msg = unescape(msg)
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("strconv.Atoi: %w", err))
	}

	me := chatRegistry.RetrieveSess(chatID)
	if me == nil {
		return s.runtimeErr(ctx, fmt.Errorf("chatRegistry.RetrieveSess: session for chat ID `%d` not found", chatID))
	}

	if errMsg, isLimited := s.checkRateLimit(ctx, me, wire.Chat, wire.ChatChannelMsgToHost); isLimited {
		return errMsg
	}

	block := wire.TLVRestBlock{}
	// the order of these TLVs matters for AIM 2.x. if out of order,
	// screen names do not appear with each chat message.
	block.Append(wire.NewTLVBE(wire.ChatTLVEnableReflectionFlag, uint8(1)))
	block.Append(wire.NewTLVBE(wire.ChatTLVSenderInformation, me.Session().TLVUserInfo()))
	block.Append(wire.NewTLVBE(wire.ChatTLVPublicWhisperFlag, []byte{}))
	block.Append(wire.NewTLVBE(wire.ChatTLVMessageInfo, wire.TLVRestBlock{
		TLVList: wire.TLVList{
			wire.NewTLVBE(wire.ChatTLVMessageInfoText, msg),
		},
	}))
	snac := wire.SNAC_0x0E_0x05_ChatChannelMsgToHost{
		Channel:      wire.ICBMChannelMIME,
		TLVRestBlock: block,
	}
	reply, err := s.ChatService.ChannelMsgToHost(ctx, me, wire.SNACFrame{}, snac)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("ChatService.ChannelMsgToHost: %w", err))
	}

	if reply == nil {
		return s.runtimeErr(ctx, errors.New("ChatService.ChannelMsgToHost: missing response "))
	}

	switch v := reply.Body.(type) {
	case wire.SNAC_0x0E_0x06_ChatChannelMsgToClient:
		msgInfo, ok := v.Bytes(wire.ChatTLVMessageInfo)
		if !ok {
			return s.runtimeErr(ctx, errors.New("ChatService.ChannelMsgToHost: missing wire.ChatTLVMessageInfo"))
		}

		reflectMsg, err := wire.UnmarshalChatMessageText(msgInfo)
		if err != nil {
			return s.runtimeErr(ctx, fmt.Errorf("wire.UnmarshalChatMessageText: %w", err))
		}

		senderInfo, ok := v.Bytes(wire.ChatTLVSenderInformation)
		if !ok {
			return s.runtimeErr(ctx, errors.New("ChatService.ChannelMsgToHost: missing wire.ChatTLVSenderInformation"))
		}

		var userInfo wire.TLVUserInfo
		if err := wire.UnmarshalBE(&userInfo, bytes.NewReader(senderInfo)); err != nil {
			return s.runtimeErr(ctx, fmt.Errorf("wire.UnmarshalBE: %w", err))
		}

		return fmt.Sprintf("CHAT_IN:%d:%s:F:%s", chatID, userInfo.ScreenName, reflectMsg)
	default:
		return s.runtimeErr(ctx, errors.New("ChatService.ChannelMsgToHost: unexpected response"))
	}
}

// ChatWhisper handles the toc_chat_send TOC command.
//
// From the TiK documentation:
//
//	Send a message in a chat room using the chat room id from CHAT_JOIN.
//	This message is directed at only one person.
//	(Currently you DO need to add this to your UI.)
//	Remember to quote and encode the message.
//	Chat whispering is different from IMs since it is linked to a chat room,
//	and should usually be displayed in the chat room UI.
//
// Command syntax: toc_chat_whisper <Chat Room ID> <dst_user> <Message>
func (s OSCARProxy) ChatWhisper(ctx context.Context, chatRegistry *ChatRegistry, args []byte) string {
	var chatIDStr, recip, msg string
	if _, err := parseArgs(args, &chatIDStr, &recip, &msg); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	msg = unescape(msg)
	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("strconv.Atoi: %w", err))
	}

	me := chatRegistry.RetrieveSess(chatID)
	if me == nil {
		return s.runtimeErr(ctx, fmt.Errorf("chatRegistry.RetrieveSess: session for chat ID `%d` not found", chatID))
	}

	if errMsg, isLimited := s.checkRateLimit(ctx, me, wire.Chat, wire.ChatChannelMsgToHost); isLimited {
		return errMsg
	}

	block := wire.TLVRestBlock{}
	block.Append(wire.NewTLVBE(wire.ChatTLVSenderInformation, me.Session().TLVUserInfo()))
	block.Append(wire.NewTLVBE(wire.ChatTLVWhisperToUser, recip))
	block.Append(wire.NewTLVBE(wire.ChatTLVMessageInfo, wire.TLVRestBlock{
		TLVList: wire.TLVList{
			wire.NewTLVBE(wire.ChatTLVMessageInfoText, msg),
		},
	}))
	snac := wire.SNAC_0x0E_0x05_ChatChannelMsgToHost{
		Channel:      wire.ICBMChannelMIME,
		TLVRestBlock: block,
	}
	if _, err = s.ChatService.ChannelMsgToHost(ctx, me, wire.SNACFrame{}, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("ChatService.ChannelMsgToHost: %w", err))
	}

	return ""
}

// ChatAccept handles the toc_chat_accept TOC command.
//
// From the TiK documentation: Accept a CHAT_INVITE message from TOC.
// The server will send a CHAT_JOIN in response.
//
// Command syntax: toc_chat_accept <Chat Room ID>
func (s OSCARProxy) ChatAccept(ctx context.Context, me *state.SessionInstance, chatRegistry *ChatRegistry, args []byte) (int, string) {
	var chatIDStr string
	if _, err := parseArgs(args, &chatIDStr); err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	chatID, err := strconv.Atoi(chatIDStr)
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("strconv.Atoi: %w", err))
	}

	chatInfo, found := chatRegistry.LookupRoom(chatID)
	if !found {
		return 0, s.runtimeErr(ctx, fmt.Errorf("chatRegistry.LookupRoom: no chat found for ID %d", chatID))
	}

	if msg, isLimited := s.checkRateLimit(ctx, me, wire.ChatNav, wire.ChatNavRequestRoomInfo); isLimited {
		return 0, msg
	}

	reqRoomSNAC := wire.SNAC_0x0D_0x04_ChatNavRequestRoomInfo{
		Cookie:         chatInfo.Cookie,
		Exchange:       chatInfo.Exchange,
		InstanceNumber: chatInfo.Instance,
	}
	reqRoomReply, err := s.ChatNavService.RequestRoomInfo(ctx, wire.SNACFrame{}, reqRoomSNAC)
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("ChatNavService.RequestRoomInfo: %w", err))
	}

	reqRoomReplyBody, ok := reqRoomReply.Body.(wire.SNAC_0x0D_0x09_ChatNavNavInfo)
	if !ok {
		return 0, s.runtimeErr(ctx, fmt.Errorf("chatNavService.RequestRoomInfo: unexpected response type %v", reqRoomReplyBody))
	}

	b, hasInfo := reqRoomReplyBody.Bytes(wire.ChatNavTLVRoomInfo)
	if !hasInfo {
		return 0, s.runtimeErr(ctx, errors.New("reqRoomReplyBody.Bytes: missing wire.ChatNavTLVRoomInfo"))
	}

	roomInfo := wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate{}
	if err := wire.UnmarshalBE(&roomInfo, bytes.NewReader(b)); err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("wire.UnmarshalBE: %w", err))
	}

	roomName, hasName := roomInfo.Bytes(wire.ChatRoomTLVRoomName)
	if !hasName {
		return 0, s.runtimeErr(ctx, errors.New("roomInfo.Bytes: missing wire.ChatRoomTLVRoomName"))
	}

	if msg, isLimited := s.checkRateLimit(ctx, me, wire.OService, wire.OServiceServiceRequest); isLimited {
		return 0, msg
	}

	svcReqSNAC := wire.SNAC_0x01_0x04_OServiceServiceRequest{
		FoodGroup: wire.Chat,
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(0x01, wire.SNAC_0x01_0x04_TLVRoomInfo{
					Cookie: chatInfo.Cookie,
				}),
			},
		},
	}
	svcReqReply, err := s.OServiceService.ServiceRequest(ctx, wire.BOS, me, wire.SNACFrame{}, svcReqSNAC, config.Listener{})
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("OServiceServiceBOS.ServiceRequest: %w", err))
	}

	svcReqReplyBody, ok := svcReqReply.Body.(wire.SNAC_0x01_0x05_OServiceServiceResponse)
	if !ok {
		return 0, s.runtimeErr(
			ctx,
			fmt.Errorf("OServiceServiceBOS.ServiceRequest: unexpected response type %v", svcReqReplyBody),
		)
	}

	loginCookie, hasCookie := svcReqReplyBody.Bytes(wire.OServiceTLVTagsLoginCookie)
	if !hasCookie {
		return 0, s.runtimeErr(ctx, errors.New("missing wire.OServiceTLVTagsLoginCookie"))
	}

	// TODO: naming for cookie: login cookie, server cookie, or auth cookie?
	serverCookie, err := s.AuthService.CrackCookie(loginCookie)
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("AuthService.RegisterChatSession: %w", err))
	}

	chatSess, err := s.AuthService.RegisterChatSession(ctx, serverCookie)
	if err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("AuthService.RegisterChatSession: %w", err))
	}

	if msg, isLimited := s.checkRateLimit(ctx, me, wire.OService, wire.OServiceClientOnline); isLimited {
		return 0, msg
	}

	if err := s.OServiceService.ClientOnline(ctx, wire.Chat, wire.SNAC_0x01_0x02_OServiceClientOnline{}, chatSess); err != nil {
		return 0, s.runtimeErr(ctx, fmt.Errorf("OServiceServiceChat.ClientOnline: %w", err))
	}

	chatRegistry.RegisterSess(chatID, chatSess)
	return chatID, fmt.Sprintf("CHAT_JOIN:%d:%s", chatID, roomName)
}

// SetConfig handles the toc_set_config TOC command.
//
// From the TiK documentation:
//
//	Set the config information for this user.
//	The config information is line oriented with the first character being the item type,
//	followed by a space, with the rest of the line being the item value.
//	Only letters, numbers, and spaces should be used.
//	Remember you will have to enclose the entire config in quotes.
//
//	Item Types:
//	  - g - Buddy Group (All Buddies until the next g or the end of config are in this group.)
//	  - b - A Buddy
//	  - p - Person on permit list
//	  - d - Person on deny list
//	  - m - Permit/Deny Mode. Possible values are
//	  - 1 - Permit All
//	  - 2 - Deny All
//	  - 3 - Permit Some
//	  - 4 - Deny Some
//
// This method doesn't attempt to validate any of the configuration,
// it saves the config as received from the client.
//
// Command syntax: toc_set_config <Config Info>
func (s OSCARProxy) SetConfig(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if status := me.Session().EvaluateRateLimit(time.Now(), 1); status == wire.RateLimitStatusLimited {
		return rateLimitExceededErr
	}

	// most TOC clients don't quote the config info argument,
	// despite what the documentation specifies.
	// this makes the argument payload incompatible for CSV parsing.
	// since this command takes a single argument,
	// we can get away with trimming quotes and spaces from the
	// byte slice before passing it to the config store.
	args = bytes.Trim(args, "'\" ")
	config := string(args)
	if config == "" {
		return s.runtimeErr(ctx, fmt.Errorf("empty config"))
	}

	if err := s.TOCConfigStore.SetTOCConfig(ctx, me.IdentScreenName(), config); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("TOCConfigStore.SaveTOCConfig: %w", err))
	}

	return ""
}

// SetInfo handles the toc_set_info TOC command.
//
// From the TiK documentation:
//
//	Set the LOCATE user information.
//	This is basic HTML.
//	Remember to encode the info.
//
// Command syntax: toc_set_info <info information>
func (s OSCARProxy) SetInfo(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if errMsg, isLimited := s.checkRateLimit(ctx, me, wire.Locate, wire.LocateSetInfo); isLimited {
		return errMsg
	}

	var info string
	if _, err := parseArgs(args, &info); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	info = unescape(info)
	snac := wire.SNAC_0x02_0x04_LocateSetInfo{
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.LocateTLVTagsInfoSigData, info),
			},
		},
	}
	if err := s.LocateService.SetInfo(ctx, me, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("LocateService.SetInfo: %w", err))
	}

	return ""
}

// SetAway handles the toc_chat_join TOC command.
//
// From the TiK documentation:
//
//	If the away message is present,
//	then the unavailable status flag is set for	the user.
//	If the away message is not present,
//	then the unavailable status flag is unset.
//	The away message is basic HTML, remember to encode the information.
//
// Command syntax: toc_set_away [<away message>]
func (s OSCARProxy) SetAway(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if errMsg, isLimited := s.checkRateLimit(ctx, me, wire.Locate, wire.LocateSetInfo); isLimited {
		return errMsg
	}

	maybeMsg, err := parseArgs(args)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	var msg string
	if len(maybeMsg) > 0 {
		msg = unescape(maybeMsg[0])
	}

	snac := wire.SNAC_0x02_0x04_LocateSetInfo{
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.LocateTLVTagsInfoUnavailableData, msg),
			},
		},
	}

	if err := s.LocateService.SetInfo(ctx, me, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("LocateService.SetInfo: %w", err))
	}

	return ""
}

// SetIdle handles the toc_set_idle TOC command.
//
// From the TiK documentation:
//
//	Set idle information.
//	If <idle secs> is 0 then the user isn't idle at all.
//	If <idle secs> is greater than 0 then the user has already been idle for <idle secs> number of seconds.
//	The server will automatically keep incrementing this number,
//	so do not repeatedly call with new idle times.
//
// Command syntax: toc_set_idle <idle secs>
func (s OSCARProxy) SetIdle(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if errMsg, isLimited := s.checkRateLimit(ctx, me, wire.OService, wire.OServiceIdleNotification); isLimited {
		return errMsg
	}

	var idleTimeStr string
	if _, err := parseArgs(args, &idleTimeStr); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	time, err := strconv.Atoi(idleTimeStr)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("strconv.Atoi: %w", err))
	}

	snac := wire.SNAC_0x01_0x11_OServiceIdleNotification{
		IdleTime: uint32(time),
	}
	if err := s.OServiceService.IdleNotification(ctx, me, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("OServiceServiceBOS.IdleNotification: %w", err))
	}

	return ""
}

// SetCaps handles the toc_set_caps TOC command.
//
// From the TiK documentation:
//
//	Set my capabilities.
//	All capabilities that we support need to be sent at the same time.
//	Capabilities are represented by UUIDs.
//
// This method automatically adds the "chat" capability since it doesn't seem to
// be sent explicitly by the official clients,
// even though they support chat.
//
// Command syntax: toc_set_caps [ <Capability 1> [<Capability 2> [...]]]
func (s OSCARProxy) SetCaps(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if errMsg, isLimited := s.checkRateLimit(ctx, me, wire.Locate, wire.LocateSetInfo); isLimited {
		return errMsg
	}

	params, err := parseArgs(args)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	caps := make([]uuid.UUID, 0, 16*(len(params)+1))
	for _, capStr := range params {
		uid, err := uuid.Parse(capStr)
		if err != nil {
			return s.runtimeErr(ctx, fmt.Errorf("UUID.Parse: %w", err))
		}
		caps = append(caps, uid)
	}

	// assume client supports chat,
	// although we may want to do this according to client ID
	caps = append(caps, wire.CapChat)
	snac := wire.SNAC_0x02_0x04_LocateSetInfo{
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.LocateTLVTagsInfoCapabilities, caps),
			},
		},
	}
	if err := s.LocateService.SetInfo(ctx, me, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("LocateService.SetInfo: %w", err))
	}

	return ""
}

// SetDir handles the toc_set_dir TOC command.
//
// From the TiK documentation:
//
//	Set the DIR user information.
//	This is a colon separated fields as in:
//
//	  "first name":"middle name":"last name":"maiden name":"city":"state":"country":"email":"allow web searches".
//
//	Should return a DIR_STATUS msg.
//	Having anything in the "allow web searches"
//	field allows people to useweb-searches to find your directory info.
//	Otherwise, they'd have to use the client.
//
// The fields "email" and "allow web searches" are ignored by this method.
//
// Command syntax: toc_set_dir <info information>
func (s OSCARProxy) SetDir(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if errMsg, isLimited := s.checkRateLimit(ctx, me, wire.Locate, wire.LocateSetDirInfo); isLimited {
		return errMsg
	}

	var info string
	if _, err := parseArgs(args, &info); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	var finalFields [9]string
	info = unescape(info)
	rawFields := strings.Split(info, ":")
	if len(rawFields) > len(finalFields) {
		return s.runtimeErr(ctx, fmt.Errorf("expected at most %d params, got %d", len(finalFields), len(rawFields)))
	}

	for i, a := range rawFields {
		finalFields[i] = strings.Trim(a, "\"")
	}

	snac := wire.SNAC_0x02_0x09_LocateSetDirInfo{
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.ODirTLVFirstName, finalFields[0]),
				wire.NewTLVBE(wire.ODirTLVMiddleName, finalFields[1]),
				wire.NewTLVBE(wire.ODirTLVLastName, finalFields[2]),
				wire.NewTLVBE(wire.ODirTLVMaidenName, finalFields[3]),
				wire.NewTLVBE(wire.ODirTLVCountry, finalFields[6]),
				wire.NewTLVBE(wire.ODirTLVState, finalFields[5]),
				wire.NewTLVBE(wire.ODirTLVCity, finalFields[4]),
			},
		},
	}
	if _, err := s.LocateService.SetDirInfo(ctx, me, wire.SNACFrame{}, snac); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("LocateService.SetDirInfo: %w", err))
	}

	return ""
}

// SendIM handles the toc_send_im TOC command.
//
// From the TiK documentation:
//
//	Send a message to a remote user.
//	Remember to quote and encode the message.
//	If the optional string "auto" is the last argument,
//	then the auto response flag will be turned on for the IM.
//
// Command syntax: toc_send_im <Destination User> <Message> [auto]
func (s OSCARProxy) SendIM(ctx context.Context, sender *state.SessionInstance, args []byte) string {
	if msg, isLimited := s.checkRateLimit(ctx, sender, wire.ICBM, wire.ICBMChannelMsgToHost); isLimited {
		return msg
	}

	var recip, msg string
	autoReply, err := parseArgs(args, &recip, &msg)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	msg = unescape(msg)
	frags, err := wire.ICBMFragmentList(msg)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("wire.ICBMFragmentList: %w", err))
	}

	snac := wire.SNAC_0x04_0x06_ICBMChannelMsgToHost{
		ChannelID:  wire.ICBMChannelIM,
		ScreenName: recip,
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.ICBMTLVAOLIMData, frags),
			},
		},
	}
	if len(autoReply) > 0 && autoReply[0] == "auto" {
		snac.Append(wire.NewTLVBE(wire.ICBMTLVAutoResponse, []byte{}))
	}

	// send message and ignore response since there is no TOC error code to
	// handle errors such as "user is offline", etc.
	_, err = s.ICBMService.ChannelMsgToHost(ctx, sender, wire.SNACFrame{}, snac)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("ICBMService.ChannelMsgToHost: %w", err))
	}

	return ""
}

// GetInfoURL handles the toc_get_info TOC command.
//
// From the TiK documentation:
//
//	Gets a user's info a GOTO_URL or ERROR message will be sent back to the client.
//
// Command syntax: toc_get_info <username>
func (s OSCARProxy) GetInfoURL(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if status := me.Session().EvaluateRateLimit(time.Now(), 1); status == wire.RateLimitStatusLimited {
		return rateLimitExceededErr
	}

	var user string
	if _, err := parseArgs(args, &user); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	cookie, err := s.newHTTPAuthToken(me.IdentScreenName())
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("newHTTPAuthToken: %w", err))
	}

	p := url.Values{}
	p.Add("cookie", cookie)
	p.Add("from", me.IdentScreenName().String())
	p.Add("user", user)
	return fmt.Sprintf("GOTO_URL:profile:info?%s", p.Encode())
}

// GetDirSearchURL handles the toc_dir_search TOC command.
//
// From the TiK documentation:
//
//	Perform a search of the Oscar Directory, using colon separated fields as in:
//
//	  "first name":"middle name":"last name":"maiden name":"city":"state":"country":"email"
//
// You can search by keyword by setting search terms in the 11th position
// (this feature is not in the TiK docs but is present in the code):
//
//	::::::::::"search kw"
//
//	Returns either a GOTO_URL or ERROR msg.
//
// Command syntax: toc_dir_search <info information>
func (s OSCARProxy) GetDirSearchURL(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if status := me.Session().EvaluateRateLimit(time.Now(), 1); status == wire.RateLimitStatusLimited {
		return rateLimitExceededErr
	}

	var info string
	if _, err := parseArgs(args, &info); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	info = unescape(info)
	params := strings.Split(info, ":")
	labels := []string{
		"first_name",
		"middle_name",
		"last_name",
		"maiden_name",
		"city",
		"state",
		"country",
		"email",
		"nop", // unused placeholder
		"nop",
		"keyword",
	}

	// map labels to param values at their corresponding positions
	p := url.Values{}
	for i, param := range params {
		if i >= len(labels) {
			break
		} else if param == "" {
			continue
		}

		p.Add(labels[i], strings.Trim(param, "\""))
	}

	if len(p) == 0 {
		return s.runtimeErr(ctx, errors.New("no search fields found"))
	}

	cookie, err := s.newHTTPAuthToken(me.IdentScreenName())
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("newHTTPAuthToken: %w", err))
	}

	p.Add("cookie", cookie)
	return fmt.Sprintf("GOTO_URL:search results:dir_search?%s", p.Encode())
}

// GetDirURL handles the toc_get_dir TOC command.
//
// From the TiK documentation:
//
//	Gets a user's dir info a GOTO_URL or ERROR message will be sent back to the client.
//
// Command syntax: toc_get_dir <username>
func (s OSCARProxy) GetDirURL(ctx context.Context, me *state.SessionInstance, args []byte) string {
	if status := me.Session().EvaluateRateLimit(time.Now(), 1); status == wire.RateLimitStatusLimited {
		return rateLimitExceededErr
	}

	var user string
	if _, err := parseArgs(args, &user); err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("parseArgs: %w", err))
	}

	cookie, err := s.newHTTPAuthToken(me.IdentScreenName())
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("newHTTPAuthToken: %w", err))
	}

	p := url.Values{}
	p.Add("cookie", cookie)
	p.Add("user", user)
	return fmt.Sprintf("GOTO_URL:directory info:dir_info?%s", p.Encode())
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

// newHTTPAuthToken creates a HMAC token for authenticating TOC HTTP requests.
func (s OSCARProxy) newHTTPAuthToken(me state.IdentScreenName) (string, error) {
	cookie, err := s.CookieBaker.Issue([]byte(me.String()))
	if err != nil {
		return "", err
	}

	// trim padding so that gaim doesn't choke on the long value
	cookie = bytes.TrimRight(cookie, "\x00")
	return hex.EncodeToString(cookie), nil
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

// unescape removes escaping from the following TOC characters: \ { } ( ) [ ] $ "
func unescape(encoded string) string {
	if !strings.ContainsRune(encoded, '\\') {
		return encoded
	}

	var escaped bool
	var result strings.Builder
	result.Grow(len(encoded))
	for i := 0; i < len(encoded); i++ {
		ch := encoded[i]
		if escaped {
			// append escaped character without the backslash
			result.WriteByte(ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else {
			result.WriteByte(ch)
		}
	}

	return result.String()
}
