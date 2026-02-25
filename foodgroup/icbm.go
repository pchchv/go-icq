package foodgroup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
	"golang.org/x/net/html"
)

const (
	warningDecayPct   = -50
	rateDecayInterval = 5 * time.Minute
)

// ICBMService provides functionality for the ICBM food group,
// which is responsible for sending and receiving instant messages and
// associated functionality such as warning, typing events, etc.
type ICBMService struct {
	relationshipFetcher   RelationshipFetcher
	buddyBroadcaster      buddyBroadcaster
	messageRelayer        MessageRelayer
	offlineMessageSaver   OfflineMessageManager
	userManager           UserManager
	feedbagManager        FeedbagManager
	timeNow               func() time.Time
	sessionRetriever      SessionRetriever
	snacRateLimits        wire.SNACRateLimits
	convoTracker          *convoTracker
	logger                *slog.Logger
	interval              time.Duration
	offlineMessageManager OfflineMessageManager
}

// NewICBMService returns a new instance of ICBMService.
func NewICBMService(
	bartItemManager BARTItemManager,
	messageRelayer MessageRelayer,
	offlineMessageSaver OfflineMessageManager,
	relationshipFetcher RelationshipFetcher,
	sessionRetriever SessionRetriever,
	userManager UserManager,
	feedbagManager FeedbagManager,
	snacRateLimits wire.SNACRateLimits,
	logger *slog.Logger,
) *ICBMService {
	return &ICBMService{
		relationshipFetcher:   relationshipFetcher,
		buddyBroadcaster:      newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		messageRelayer:        messageRelayer,
		offlineMessageSaver:   offlineMessageSaver,
		offlineMessageManager: offlineMessageSaver,
		userManager:           userManager,
		feedbagManager:        feedbagManager,
		timeNow:               time.Now,
		sessionRetriever:      sessionRetriever,
		snacRateLimits:        snacRateLimits,
		convoTracker:          newConvoTracker(),
		logger:                logger,
		interval:              rateDecayInterval,
	}
}

// RestoreWarningLevel restores the warning level from the last stored value at login time,
// accounting for time passed between logins.
func (s ICBMService) RestoreWarningLevel(ctx context.Context, instance *state.SessionInstance) error {
	u, err := s.userManager.User(ctx, instance.IdentScreenName())
	if err != nil {
		return errors.New("failed to get user: " + err.Error())
	} else if u == nil {
		return state.ErrNoUser
	} else if u.LastWarnLevel == 0 {
		// user had no warning at the end of last session
		return nil
	}

	// get the rate class for sending IMs, which gets limited when the user gets warned
	classID, ok := s.snacRateLimits.RateClassLookup(wire.ICBM, wire.ICBMChannelMsgToHost)
	if !ok {
		panic("failed to retrieve rate class for ICBMChannelMsgToHost")
	}

	// increment warning level by the amount of time that has passed since last
	// login, proportionally increasing the warning level
	warnDelta := calcElapsedWarningLevel(u.LastWarnUpdate, s.timeNow(), s.interval)
	newWarning := int16(u.LastWarnLevel) + warnDelta
	instance.Session().SetWarning(0)
	instance.Session().ScaleWarningAndRateLimit(newWarning, classID)
	if instance.Warning() > 0 {
		s.logger.DebugContext(
			ctx, "restored warning level with time decay applied since last login",
			"stored_level", u.LastWarnLevel,
			"time_since_update", s.timeNow().Sub(u.LastWarnUpdate),
			"decay_delta", warnDelta,
			"final_level", instance.Warning(),
		)
	} else {
		s.logger.DebugContext(
			ctx, "warning level decayed to zero since last login",
			"stored_level", u.LastWarnLevel,
			"time_since_update", s.timeNow().Sub(u.LastWarnUpdate),
			"decay_delta", warnDelta,
		)
	}

	return nil
}

func (s ICBMService) sendOfflineMessage(ctx context.Context, instance *state.SessionInstance, inFrame wire.SNACFrame, inBody wire.SNAC_0x04_0x06_ICBMChannelMsgToHost) (*wire.SNACMessage, error) {
	recip := state.NewIdentScreenName(inBody.ScreenName)
	offlineMsg := state.OfflineMessage{
		Message:   inBody,
		Recipient: recip,
		Sender:    instance.IdentScreenName(),
		Sent:      s.timeNow().UTC(),
	}
	if _, err := s.offlineMessageSaver.SaveMessage(ctx, offlineMsg); err != nil {
		if errors.Is(err, state.ErrOfflineInboxFull) {
			return newICBMErr(
				inFrame.RequestID,
				wire.ErrorCodeNotLoggedOn,
				wire.NewTLVBE(wire.ErrorTLVErrorSubcode, wire.ICBMSubErrOfflineIMExceedMax),
			), nil
		}
		return nil, errors.New("save ICBM offline message failed: " + err.Error())
	}

	if instance.UIN() > 0 {
		return newICBMErr(inFrame.RequestID, wire.ErrorCodeNotLoggedOn), nil
	}

	if _, requestedConfirmation := inBody.TLVRestBlock.Bytes(wire.ICBMTLVRequestHostAck); requestedConfirmation {
		// ack message back to sender
		return &wire.SNACMessage{
			Frame: wire.SNACFrame{
				FoodGroup: wire.ICBM,
				SubGroup:  wire.ICBMHostAck,
				RequestID: inFrame.RequestID,
			},
			Body: wire.SNAC_0x04_0x0C_ICBMHostAck{
				Cookie:     inBody.Cookie,
				ChannelID:  inBody.ChannelID,
				ScreenName: inBody.ScreenName,
			},
		}, nil
	}

	return nil, nil
}

// ringBuffer is a fixed-size circular buffer with 3 slots for storing time values.
type ringBuffer struct {
	cur  int          // Current cursor position (0, 1, or 2).
	vals [3]time.Time // Fixed-size array to store time values.
}

// set stores the given time at the current cursor position and advances the cursor.
func (r *ringBuffer) set(v time.Time) {
	r.vals[r.cur] = v
	r.cur = (r.cur + 1) % len(r.vals)
}

// val returns the time at the current cursor position.
func (r *ringBuffer) val() time.Time {
	return r.vals[r.cur]
}

// convoTracker keeps track of messages initiated from a sender to a recipient.
// A user (the warner) can only warn another user (the warnee)
// only if the warner has received a message from the warnee.
// The warner may only warn 1 time per message received from warnee.
// The warner may only warn the warnee up to 3 times per warn window.
type convoTracker struct {
	convos *cache.Cache
	warns  *cache.Cache
	window time.Duration
}

func newConvoTracker() *convoTracker {
	window := 1 * time.Hour
	return &convoTracker{
		convos: cache.New(window, window),
		warns:  cache.New(window, window),
		window: window,
	}
}

func (w *convoTracker) key(sender state.IdentScreenName, recip state.IdentScreenName) string {
	return sender.String() + recip.String()
}

// trackWarn attempts to record a warning from warner to warnee.
// It returns true if the warning is allowed
// (warnee has sent more messages than warnings in the current window),
// or false if the warning limit has been reached or no conversation exists in the current window.
func (w *convoTracker) trackWarn(now time.Time, warner, warnee state.IdentScreenName) bool {
	key := w.key(warnee, warner)
	convos, found := w.convos.Get(key)
	if !found {
		// no convos tracked, can't warn
		return false
	}

	// get convo count during window
	var convoCt int
	windowStart := now.Add(-w.window)
	for _, v := range convos.(*ringBuffer).vals {
		if v.After(windowStart) {
			convoCt++
		}
	}

	warns, found := w.warns.Get(key)
	if !found {
		warns = &ringBuffer{}
		w.warns.Set(key, warns, time.Hour)
	}

	// get warn count during window
	var warnCount int
	for _, v := range warns.(*ringBuffer).vals {
		if v.After(windowStart) {
			warnCount++
		}
	}

	if convoCt <= warnCount {
		return false
	}

	warns.(*ringBuffer).set(now)
	return true
}

// trackConvo records a conversation from sender to recipient at the given time.
func (w *convoTracker) trackConvo(now time.Time, sender, recip state.IdentScreenName) {
	k := w.key(sender, recip)
	buf, found := w.convos.Get(k)
	if !found {
		buf = &ringBuffer{}
		w.convos.Set(k, buf, time.Hour)
	}

	buf.(*ringBuffer).set(now)
}

func newICBMErr(requestID uint32, errCode uint16, tlvs ...wire.TLV) *wire.SNACMessage {
	body := wire.SNACError{
		Code: errCode,
	}
	if len(tlvs) > 0 {
		body.AppendList(tlvs)
	}

	return &wire.SNACMessage{
		Frame: wire.SNACFrame{
			FoodGroup: wire.ICBM,
			SubGroup:  wire.ICBMErr,
			RequestID: requestID,
		},
		Body: body,
	}
}

func calcElapsedWarningLevel(lastWarnUpdate time.Time, now time.Time, interval time.Duration) (warnDelta int16) {
	// time passed since last signoff
	since := now.Sub(lastWarnUpdate)
	// how many times warning decayed since last signoff
	decayPeriods := int(since / interval)
	// total amount warning decreased since last signoff
	warnDelta = int16(decayPeriods * warningDecayPct)
	return
}

// addExternalIP appends the client's IP address to the TLV if it's an ICBM rendezvous proposal/accept message.
func addExternalIP(instance *state.SessionInstance, tlv wire.TLV) (wire.TLV, error) {
	frag := wire.ICBMCh2Fragment{}
	if err := wire.UnmarshalBE(&frag, bytes.NewReader(tlv.Value)); err != nil {
		return tlv, errors.New("wire.UnmarshalBE: " + err.Error())
	}

	if frag.Type != wire.ICBMRdvMessagePropose {
		return tlv, nil
	}

	if frag.HasTag(wire.ICBMRdvTLVTagsRequesterIP) && instance.RemoteAddr() != nil && instance.RemoteAddr().Addr().Is4() {
		ip := instance.RemoteAddr().Addr()
		// replace the IP set by the client with the actual IP seen by the
		// server. unlike AOL’s original behavior, this allows NATed clients
		// to use rendezvous by replacing their LAN IP with the correct
		// external IP.
		frag.Replace(wire.NewTLVBE(wire.ICBMRdvTLVTagsRequesterIP, ip.AsSlice()))
		// append the client’s IP as seen by the server. the recipient uses
		// this to verify that the sender’s claimed IP matches what the server
		// detects. although redundant since we override the requester IP
		// above, it remains required for client compatibility.
		frag.Append(wire.NewTLVBE(wire.ICBMRdvTLVTagsVerifiedIP, ip.AsSlice()))
		return wire.NewTLVBE(tlv.Tag, frag), nil
	}

	return tlv, nil
}

// stripHTML extracts plaintext from HTML content.
func stripHTML(text []byte) []byte {
	if len(text) == 0 {
		return text
	}

	var result strings.Builder
	tok := html.NewTokenizer(strings.NewReader(string(text)))
	for {
		switch tok.Next() {
		case html.TextToken:
			result.Write(tok.Text())
		case html.SelfClosingTagToken, html.StartTagToken:
			tn, _ := tok.TagName()
			if string(tn) == "br" {
				result.WriteByte('\n')
			}
		case html.ErrorToken:
			if tok.Err() == io.EOF {
				return []byte(result.String())
			}

			// on error return what we have
			return []byte(result.String())
		}
	}
}

// stripHTMLFromICBMTLV transforms an ICBMTLVAOLIMData TLV by
// stripping HTML from the message text for
// clients that don't support XHTML.
func stripHTMLFromICBMTLV(tlv wire.TLV) (wire.TLV, error) {
	var frags []wire.ICBMCh1Fragment
	if err := wire.UnmarshalBE(&frags, bytes.NewBuffer(tlv.Value)); err != nil {
		return tlv, errors.New("unmarshal ICBM fragments: " + err.Error())
	}

	var modified bool
	for i, frag := range frags {
		if frag.ID == 1 { // 1 = message text
			msg := wire.ICBMCh1Message{}
			if wire.UnmarshalBE(&msg, bytes.NewBuffer(frag.Payload)) == nil {
				// strip HTML from message text
				strippedText := stripHTML(msg.Text)
				if !bytes.Equal(strippedText, msg.Text) {
					msg.Text = strippedText
					// remarshal the message
					msgBuf := bytes.Buffer{}
					if wire.MarshalBE(msg, &msgBuf) == nil {
						frags[i].Payload = msgBuf.Bytes()
						modified = true
					}
				}
			}
		}
	}

	if !modified {
		return tlv, nil
	}

	// remarshal the fragments
	newValue, err := wire.MarshalICBMFragmentList(frags)
	if err != nil {
		return tlv, err
	}

	return wire.NewTLVBE(tlv.Tag, newValue), nil
}
