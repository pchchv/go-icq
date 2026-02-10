package toc

import (
	"fmt"
	"strings"

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
