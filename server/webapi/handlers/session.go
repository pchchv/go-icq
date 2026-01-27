package handlers

import (
	"context"
	"encoding/xml"
	"log/slog"

	"github.com/google/uuid"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// Buddy represents a buddy in the buddy list.
type Buddy struct {
	AimID     string `json:"aimId"`
	State     string `json:"state"`
	AwayMsg   string `json:"awayMsg,omitempty"`
	UserType  string `json:"userType"`
	StatusMsg string `json:"statusMsg,omitempty"`
}

// BuddyGroup represents a group of buddies.
type BuddyGroup struct {
	Name    string  `json:"name"`
	Buddies []Buddy `json:"buddies"`
}

// BuddyListService defines methods for buddy list operations.
type BuddyListService interface {
	GetBuddyList(ctx context.Context, screenName state.IdentScreenName) ([]BuddyGroup, error)
}

// BuddyListRegistry defines methods for buddy list management.
type BuddyListRegistry interface {
	RegisterBuddyList(ctx context.Context, screenName state.IdentScreenName) error
	UnregisterBuddyList(ctx context.Context, screenName state.IdentScreenName) error
}

// AuthService defines methods needed for authentication.
type AuthService interface {
	BUCPChallenge(ctx context.Context, bodyIn wire.SNAC_0x17_0x06_BUCPChallengeRequest, newUUID func() uuid.UUID) (wire.SNACMessage, error)
	BUCPLogin(ctx context.Context, bodyIn wire.SNAC_0x17_0x02_BUCPLoginRequest, newUserFn func(screenName state.DisplayScreenName) (state.User, error), advertisedHost string) (wire.SNACMessage, error)
	RegisterBOSSession(ctx context.Context, authCookie state.ServerCookie) (*state.SessionInstance, error)
}

// SessionManager defines methods for OSCAR session management.
type SessionManager interface {
	AddSession(ctx context.Context, screenName state.DisplayScreenName) (*state.SessionInstance, error)
	RemoveSession(instance *state.SessionInstance)
	RelayToScreenName(ctx context.Context, screenName state.IdentScreenName, msg wire.SNACMessage)
}

// StartSessionXMLResponse represents the XML response for startSession endpoint.
type StartSessionXMLResponse struct {
	XMLName    xml.Name `xml:"response"`
	StatusCode int      `xml:"statusCode"`
	StatusText string   `xml:"statusText"`
	Data       struct {
		AimSID          string `xml:"aimsid"`
		FetchTimeout    int    `xml:"fetchTimeout"`
		TimeToNextFetch int    `xml:"timeToNextFetch"`
		FetchBaseURL    string `xml:"fetchBaseURL"`
		WellKnownUrls   *struct {
			WebApiBase   string `xml:"webApiBase"`
			FetchBaseURL string `xml:"fetchBaseURL"`
		} `xml:"wellKnownUrls,omitempty"`
		MyInfo *struct {
			AimID     string `xml:"aimId"`
			DisplayID string `xml:"displayId"`
			Buddylist struct {
				Groups *[]BuddyGroup `xml:"group,omitempty"`
			} `xml:"buddylist,omitempty"`
		} `xml:"myInfo,omitempty"`
		Events *struct {
			BuddyList struct {
				Groups *[]BuddyGroup `xml:"group,omitempty"`
			} `xml:"buddylist"`
		} `xml:"events,omitempty"`
	} `xml:"data"`
}

// StartSessionResponse represents the response for startSession endpoint.
type StartSessionResponse struct {
	Response struct {
		StatusCode int    `json:"statusCode"`
		StatusText string `json:"statusText"`
		Data       struct {
			AimSID          string                 `json:"aimsid"`
			FetchTimeout    int                    `json:"fetchTimeout"`
			TimeToNextFetch int                    `json:"timeToNextFetch"`
			FetchBaseURL    string                 `json:"fetchBaseURL"` // Gromit expects this directly in data!
			Events          map[string]interface{} `json:"events,omitempty"`
			WellKnownUrls   map[string]string      `json:"wellKnownUrls,omitempty"`
		} `json:"data"`
	} `json:"response"`
}

// EndSessionResponse represents the response for endSession endpoint.
type EndSessionResponse struct {
	Response struct {
		StatusCode int    `json:"statusCode"`
		StatusText string `json:"statusText"`
	} `json:"response"`
}

// SessionHandler handles Web AIM API session management endpoints.
type SessionHandler struct {
	OSCARSessionManager SessionManager
	BuddyListRegistry   BuddyListRegistry
	OSCARAuthService    AuthService
	BuddyListService    BuddyListService
	BuddyBroadcaster    BuddyBroadcaster
	BuddyListManager    *BuddyListManager
	SessionManager      *state.WebAPISessionManager
	TokenStore          TokenStore
	Logger              *slog.Logger
}
