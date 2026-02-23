package foodgroup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"regexp"
	"strconv"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
	"golang.org/x/net/html"
)

// rollDiceRgxp matches a roll dice chat command.
// ex: //roll //roll-sides3 //roll-dice2 //role-sides3-dice2
var rollDiceRgxp = regexp.MustCompile(`^//roll(?:-(dice|sides)([0-9]{1,3}))?(?:-(dice|sides)([0-9]{1,3}))?\s*$`)

// ChatService provides functionality for the Chat food group,
// which is responsible for sending and receiving chat messages.
type ChatService struct {
	chatMessageRelayer ChatMessageRelayer
	randRollDie        func(sides int) int
}

// NewChatService creates a new instance of ChatService.
func NewChatService(chatMessageRelayer ChatMessageRelayer) *ChatService {
	return &ChatService{
		chatMessageRelayer: chatMessageRelayer,
		randRollDie: func(sides int) int {
			// generate random number between 1 and sides
			return rand.IntN(sides) + 1
		},
	}
}

func setOnlineChatUsers(ctx context.Context, instance *state.SessionInstance, chatMessageRelayer ChatMessageRelayer) {
	snacPayloadOut := wire.SNAC_0x0E_0x03_ChatUsersJoined{}
	sessions := chatMessageRelayer.AllSessions(instance.ChatRoomCookie())
	for _, session := range sessions {
		snacPayloadOut.Users = append(snacPayloadOut.Users, session.TLVUserInfo())
	}

	chatMessageRelayer.RelayToScreenName(ctx, instance.ChatRoomCookie(), instance.IdentScreenName(), wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Chat,
			SubGroup:  wire.ChatUsersJoined,
		},
		Body: snacPayloadOut,
	})
}

func sendChatRoomInfoUpdate(ctx context.Context, instance *state.SessionInstance, chatMessageRelayer ChatMessageRelayer, room state.ChatRoom) {
	chatMessageRelayer.RelayToScreenName(ctx, instance.ChatRoomCookie(), instance.IdentScreenName(), wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Chat,
			SubGroup:  wire.ChatRoomInfoUpdate,
		},
		Body: wire.SNAC_0x0E_0x02_ChatRoomInfoUpdate{
			Exchange:       room.Exchange(),
			Cookie:         room.Cookie(),
			InstanceNumber: room.InstanceNumber(),
			DetailLevel:    room.DetailLevel(),
			TLVBlock: wire.TLVBlock{
				TLVList: room.TLVList(),
			},
		},
	})
}

func alertUserJoined(ctx context.Context, instance *state.SessionInstance, chatMessageRelayer ChatMessageRelayer) {
	chatMessageRelayer.RelayToAllExcept(ctx, instance.ChatRoomCookie(), instance.IdentScreenName(), wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Chat,
			SubGroup:  wire.ChatUsersJoined,
		},
		Body: wire.SNAC_0x0E_0x03_ChatUsersJoined{
			Users: []wire.TLVUserInfo{
				instance.Session().TLVUserInfo(),
			},
		},
	})
}

func alertUserLeft(ctx context.Context, instance *state.SessionInstance, chatMessageRelayer ChatMessageRelayer) {
	chatMessageRelayer.RelayToAllExcept(ctx, instance.ChatRoomCookie(), instance.IdentScreenName(), wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.Chat,
			SubGroup:  wire.ChatUsersLeft,
		},
		Body: wire.SNAC_0x0E_0x04_ChatUsersLeft{
			Users: []wire.TLVUserInfo{
				instance.Session().TLVUserInfo(),
			},
		},
	})
}

func newChatTLVBlock(body wire.SNAC_0x0E_0x05_ChatChannelMsgToHost, instance *state.SessionInstance, msg any) wire.TLVRestBlock {
	block := wire.TLVRestBlock{}
	// the order of these TLVs matters for AIM 2.x. if out of order, screen
	// names do not appear with each chat message.
	block.Append(wire.NewTLVBE(wire.ChatTLVSenderInformation, instance.Session().TLVUserInfo()))
	if body.HasTag(wire.ChatTLVPublicWhisperFlag) {
		// send message to all chat room participants
		block.Append(wire.NewTLVBE(wire.ChatTLVPublicWhisperFlag, []byte{}))
	}

	block.Append(wire.NewTLVBE(wire.ChatTLVMessageInfo, msg))
	return block
}

// extractChatMessage extracts plaintext message text from
// HTML located in chat message info TLV(0x05).
func extractChatMessage(msg wire.TLVRestBlock) ([]byte, error) {
	b, hasMsg := msg.Bytes(wire.ChatTLVMessageInfoText)
	if !hasMsg {
		return nil, errors.New("SNAC(0x0E,0x05) has no chat msg text TLV")
	}

	tok := html.NewTokenizer(bytes.NewBuffer(b))
	for {
		switch tok.Next() {
		case html.TextToken:
			return tok.Text(), nil
		case html.ErrorToken:
			err := tok.Err()
			if err == io.EOF {
				err = nil
			}
			return nil, err
		}
	}
}

// parseDiceCommand gets the number of dice and sides from a die roll command.
//
// The roll command is activated with //roll followed by up to
// two arguments to indicate die count and side count.
// By default, there are 2 dice and 6 sides.
//
//   - //roll               2x 6-sided dice
//   - //roll-dice4         4x 6-sided dice
//   - //roll-sides8        2x 8-sided dice
//   - //roll-dice4-sides8  4x 8-sided dice
//
// The -dice param can not exceed 15 and -sides param cannot exceed 999.
func parseDiceCommand(in []byte) (valid bool, dice int, sides int) {
	matches := rollDiceRgxp.FindSubmatch(in)
	if len(matches) == 0 {
		return false, 0, 0
	}

	args := matches[1:]
	if len(args[0]) > 0 && bytes.Equal(args[0], args[2]) {
		// "sides" or "dice" appears twice
		return false, 0, 0
	}

	dice, sides = 2, 6
	for i := 0; i < len(args); i += 2 {
		var err error
		cmd := string(args[i])
		val := string(args[i+1])
		switch cmd {
		case "sides":
			sides, err = strconv.Atoi(val)
			if err != nil || sides == 0 || sides > 999 {
				return false, 0, 0
			}
		case "dice":
			dice, err = strconv.Atoi(val)
			if err != nil || dice == 0 || dice > 15 {
				return false, 0, 0
			}
		}
	}

	return true, dice, sides
}
