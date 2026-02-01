package webapi

import (
	"time"

	"github.com/pchchv/go-icq/server/webapi/types"
	"github.com/pchchv/go-icq/wire"
)

// TypingNotificationToWebAPIEvent converts an OSCAR typing notification to a WebAPI event.
func TypingNotificationToWebAPIEvent(notification wire.SNAC_0x04_0x14_ICBMClientEvent) types.Event {
	typing := false
	switch notification.Event {
	case 0x0002: // typing started
		typing = true
	case 0x0001: // typing stopped
		typing = false
	}

	return types.Event{
		Type:      types.EventTypeTyping,
		Timestamp: time.Now().Unix(),
		Data: types.TypingEvent{
			From:   notification.ScreenName,
			Typing: typing,
		},
	}
}
