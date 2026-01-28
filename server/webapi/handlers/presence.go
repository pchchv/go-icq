package handlers

import (
	"context"

	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// BuddyBroadcaster broadcasts buddy presence updates.
type BuddyBroadcaster interface {
	BroadcastBuddyArrived(ctx context.Context, screenName state.IdentScreenName, userInfo wire.TLVUserInfo) error
	BroadcastBuddyDeparted(ctx context.Context, instance *state.SessionInstance) error
}

// BuddyPresenceInfo represents presence information for a buddy.
type BuddyPresenceInfo struct {
	AimID      string `json:"aimId" xml:"aimId"`
	State      string `json:"state" xml:"state"` // "online", "offline", "away", "idle"
	AwayMsg    string `json:"awayMsg,omitempty" xml:"awayMsg,omitempty"`
	StatusMsg  string `json:"statusMsg,omitempty" xml:"statusMsg,omitempty"`
	OnlineTime int64  `json:"onlineTime,omitempty" xml:"onlineTime,omitempty"`
	IdleTime   int    `json:"idleTime,omitempty" xml:"idleTime,omitempty"`
	UserType   string `json:"userType" xml:"userType"` // "aim", "icq", "admin"
}
