package webapi

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/pchchv/go-icq/server/webapi/types"
	"github.com/pchchv/go-icq/state"
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

// WebAPIToICBM converts a Web API message to OSCAR ICBM format.
func WebAPIToICBM(sender state.IdentScreenName, recipient string, message string, autoResponse bool) (wire.SNAC_0x04_0x06_ICBMChannelMsgToHost, error) {
	// generate message cookie
	var cookie [8]byte
	if _, err := rand.Read(cookie[:]); err != nil {
		return wire.SNAC_0x04_0x06_ICBMChannelMsgToHost{}, err
	}

	cookieUint64 := binary.BigEndian.Uint64(cookie[:])
	// create ICBM fragment list for the message
	frags, err := wire.ICBMFragmentList(message)
	if err != nil {
		return wire.SNAC_0x04_0x06_ICBMChannelMsgToHost{}, err
	}

	// marshal the fragments
	buf := &bytes.Buffer{}
	for _, frag := range frags {
		if err := wire.MarshalBE(frag, buf); err != nil {
			return wire.SNAC_0x04_0x06_ICBMChannelMsgToHost{}, err
		}
	}

	// build ICBM message
	icbmMsg := wire.SNAC_0x04_0x06_ICBMChannelMsgToHost{
		Cookie:     cookieUint64,
		ChannelID:  wire.ICBMChannelIM,
		ScreenName: recipient,
		TLVRestBlock: wire.TLVRestBlock{
			TLVList: wire.TLVList{
				wire.NewTLVBE(wire.ICBMTLVAOLIMData, buf.Bytes()),
			},
		},
	}

	// add auto-response flag if applicable
	if autoResponse {
		icbmMsg.Append(wire.NewTLVBE(wire.ICBMTLVAutoResponse, []byte{}))
	}

	return icbmMsg, nil
}
