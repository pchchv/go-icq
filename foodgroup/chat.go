package foodgroup

import (
	"context"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

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
