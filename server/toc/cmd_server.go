package toc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

var (
	rateLimitExceededErr = "ERROR:903"
	cmdInternalSvcErr    = "ERROR:989:internal server error"
)

// ChatUpdateBuddyArrived handles the CHAT_UPDATE_BUDDY TOC command for
// chat room arrival events.
//
// From the TiK documentation:
//
//	This one command handles arrival/departs from a chat room.
//	The very first message of this type for each chat room contains the
//	users already in the room.
//
// Command syntax: CHAT_UPDATE_BUDDY:<Chat Room Id>:<Inside? T/F>:<User 1>:<User 2>...
func (s OSCARProxy) ChatUpdateBuddyArrived(snac wire.SNAC_0x0E_0x03_ChatUsersJoined, chatID int) string {
	users := make([]string, 0, len(snac.Users))
	for _, u := range snac.Users {
		users = append(users, u.ScreenName)
	}

	return fmt.Sprintf("CHAT_UPDATE_BUDDY:%d:T:%s", chatID, strings.Join(users, ":"))
}

// ChatUpdateBuddyLeft handles the CHAT_UPDATE_BUDDY TOC command for
// chat room departure events.
//
// From the TiK documentation:
//
//	This one command handles arrival/departs from a chat room.
//	The very first message of this type for each chat room contains the
//	users already in the room.
//
// Command syntax: CHAT_UPDATE_BUDDY:<Chat Room Id>:<Inside? T/F>:<User 1>:<User 2>...
func (s OSCARProxy) ChatUpdateBuddyLeft(snac wire.SNAC_0x0E_0x04_ChatUsersLeft, chatID int) string {
	users := make([]string, 0, len(snac.Users))
	for _, u := range snac.Users {
		users = append(users, u.ScreenName)
	}

	return fmt.Sprintf("CHAT_UPDATE_BUDDY:%d:F:%s", chatID, strings.Join(users, ":"))
}

// ChatIn handles the CHAT_IN TOC command.
//
// From the TiK documentation: a chat message was sent in a chat room.
//
// Command syntax: CHAT_IN:<Chat Room Id>:<Source User>:<Whisper? T/F>:<Message>
func (s OSCARProxy) ChatIn(ctx context.Context, snac wire.SNAC_0x0E_0x06_ChatChannelMsgToClient, chatID int) string {
	b, ok := snac.Bytes(wire.ChatTLVSenderInformation)
	if !ok {
		return s.runtimeErr(ctx, errors.New("snac.Bytes: missing wire.ChatTLVSenderInformation"))
	}

	u := wire.TLVUserInfo{}
	err := wire.UnmarshalBE(&u, bytes.NewReader(b))
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("wire.UnmarshalBE: %w", err))
	}

	b, ok = snac.Bytes(wire.ChatTLVMessageInfo)
	if !ok {
		return s.runtimeErr(ctx, errors.New("snac.Bytes: missing wire.ChatTLVMessageInfo"))
	}

	text, err := wire.UnmarshalChatMessageText(b)
	if err != nil {
		return s.runtimeErr(ctx, fmt.Errorf("wire.UnmarshalChatMessageText: %w", err))
	}

	return fmt.Sprintf("CHAT_IN:%d:%s:F:%s", chatID, u.ScreenName, text)
}

// RecvChat routes incoming SNAC messages from the chat server to their corresponding TOC handlers.
// It ignores any SNAC messages for which there is no TOC response.
func (s OSCARProxy) RecvChat(ctx context.Context, me *state.SessionInstance, chatID int, ch chan<- []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-me.Closed():
			return
		case snac := <-me.ReceiveMessage():
			switch v := snac.Body.(type) {
			case wire.SNAC_0x0E_0x04_ChatUsersLeft:
				sendOrCancel(ctx, ch, s.ChatUpdateBuddyLeft(v, chatID))
			case wire.SNAC_0x0E_0x03_ChatUsersJoined:
				sendOrCancel(ctx, ch, s.ChatUpdateBuddyArrived(v, chatID))
			case wire.SNAC_0x0E_0x06_ChatChannelMsgToClient:
				sendOrCancel(ctx, ch, s.ChatIn(ctx, v, chatID))
			default:
				s.Logger.DebugContext(ctx, fmt.Sprintf("unsupported snac. foodgroup: %s subgroup: %s",
					wire.FoodGroupName(snac.Frame.FoodGroup),
					wire.SubGroupName(snac.Frame.FoodGroup, snac.Frame.SubGroup)))
			}
		}
	}
}

// UpdateBuddyArrival handles the UPDATE_BUDDY TOC command for buddy arrival events.
//
// From the TiK documentation:
//
//	This one command handles arrival/depart/updates.
//	Evil Amount is a percentage, Signon Time is UNIX epoc,
//	idle time is in minutes, UC (User Class) is a two/three character string.
//	  - uc[0]
//	    - ' ' - Ignore
//	    - 'A' - On AOL
//	  - uc[1]
//	    - ' ' - Ignore
//	    - 'A' - Oscar Admin
//	    - 'U' - Oscar Unconfirmed
//	    - 'O' - Oscar Normal
//	  - uc[2]
//	    - '\0' - Ignore
//	    - ' ' - Ignore
//	    - 'U' - The user has set their unavailable flag.
//
// Command syntax: UPDATE_BUDDY:<Buddy User>:<Online? T/F>:<Evil Amount>:<Signon Time>:<IdleTime>:<UC>
func (s OSCARProxy) UpdateBuddyArrival(snac wire.SNAC_0x03_0x0B_BuddyArrived) string {
	return userInfoToUpdateBuddy(snac.TLVUserInfo)
}

// UpdateBuddyDeparted handles the UPDATE_BUDDY TOC command for buddy departure events.
//
// From the TiK documentation:
//
//	This one command handles arrival/depart/updates.
//	Evil Amount is a percentage,
//	Signon Time is UNIX epoc,
//	idle time is in minutes,
//	UC (User Class) is a two/three character string.
//	  - uc[0]
//	    - ' ' - Ignore
//	    - 'A' - On AOL
//	  - uc[1]
//	    - ' ' - Ignore
//	    - 'A' - Oscar Admin
//	    - 'U' - Oscar Unconfirmed
//	    - 'O' - Oscar Normal
//	  - uc[2]
//	    - '\0' - Ignore
//	    - ' ' - Ignore
//	    - 'U' - The user has set their unavailable flag.
//
// Command syntax: UPDATE_BUDDY:<Buddy User>:<Online? T/F>:<Evil Amount>:<Signon Time>:<IdleTime>:<UC>
func (s OSCARProxy) UpdateBuddyDeparted(snac wire.SNAC_0x03_0x0C_BuddyDeparted) string {
	return fmt.Sprintf("UPDATE_BUDDY:%s:F:0:0:0:   ", snac.ScreenName)
}

// userInfoToUpdateBuddy creates an UPDATE_BUDDY server reply from a User Info TLV.
func userInfoToUpdateBuddy(snac wire.TLVUserInfo) string {
	online, _ := snac.Uint32BE(wire.OServiceUserInfoSignonTOD)
	idle, _ := snac.Uint16BE(wire.OServiceUserInfoIdleTime)
	uc := [3]string{" ", "O", " "}
	if snac.IsAway() {
		uc[2] = "U"
	}

	class := strings.Join(uc[:], "")
	return fmt.Sprintf("UPDATE_BUDDY:%s:%s:%s:%d:%d:%s", snac.ScreenName, "T", fmt.Sprintf("%d", snac.WarningLevel/10), online, idle, class)
}

func sendOrCancel(ctx context.Context, ch chan<- []byte, msg string) {
	select {
	case <-ctx.Done():
		return
	case ch <- []byte(msg):
		return
	}
}
