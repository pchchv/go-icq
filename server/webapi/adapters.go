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

// ICBMToWebAPIEvent converts an incoming ICBM message to a WebAPI event.
func ICBMToWebAPIEvent(icbm wire.SNAC_0x04_0x07_ICBMChannelMsgToClient) (types.Event, error) {
	// extract message text
	var messageText string
	var autoResponse bool
	// check for AOL IM data
	if msgData, hasMsg := icbm.Bytes(wire.ICBMTLVAOLIMData); hasMsg {
		msgText, err := wire.UnmarshalICBMMessageText(msgData)
		if err == nil {
			messageText = msgText
		}
	}

	// check for auto-response flag
	if _, hasAutoResp := icbm.Bytes(wire.ICBMTLVAutoResponse); hasAutoResp {
		autoResponse = true
	}

	// extract sender screen name from TLVUserInfo
	var senderScreenName string
	if icbm.TLVUserInfo.ScreenName != "" {
		senderScreenName = icbm.TLVUserInfo.ScreenName
	}

	// create WebAPI event
	event := types.Event{
		Type:      types.EventTypeIM,
		Timestamp: time.Now().Unix(),
		Data: types.IMEvent{
			From:      senderScreenName,
			Message:   messageText,
			Timestamp: float64(time.Now().Unix()),
			AutoResp:  autoResponse,
		},
	}

	return event, nil
}
