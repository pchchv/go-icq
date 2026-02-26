package foodgroup

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/pchchv/go-icq/config"
	"github.com/pchchv/go-icq/state"
	"github.com/pchchv/go-icq/wire"
)

// MaxConcurrentLoginsPerUser is the
// maximum number of concurrent logins allowed for a single user.
const MaxConcurrentLoginsPerUser = 5

// AuthService provides client login and session management services.
// It supports both FLAP (AIM v1.0-v3.0) and BUCP (AIM v3.5-v5.9) authentication modes.
type AuthService struct {
	chatMessageRelayer         ChatMessageRelayer
	chatSessionRegistry        ChatSessionRegistry
	config                     config.Config
	cookieBaker                CookieBaker
	logger                     *slog.Logger
	sessionManager             SessionRegistry
	sessionRetriever           SessionRetriever
	userManager                UserManager
	accountManager             AccountManager
	bartItemManager            BARTItemManager
	rateLimitClasses           wire.RateLimitClasses
	timeNow                    func() time.Time
	maxConcurrentLoginsPerUser int
	createAccount              state.CreateAccountFunc
}

// NewAuthService creates a new instance of AuthService.
func NewAuthService(
	cfg config.Config,
	sessionManager SessionRegistry,
	sessionRetriever SessionRetriever,
	chatSessionRegistry ChatSessionRegistry,
	userManager UserManager,
	cookieBaker CookieBaker,
	chatMessageRelayer ChatMessageRelayer,
	accountManager AccountManager,
	bartItemManager BARTItemManager,
	classes wire.RateLimitClasses,
	createAccount state.CreateAccountFunc,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		chatSessionRegistry:        chatSessionRegistry,
		config:                     cfg,
		cookieBaker:                cookieBaker,
		sessionManager:             sessionManager,
		sessionRetriever:           sessionRetriever,
		userManager:                userManager,
		chatMessageRelayer:         chatMessageRelayer,
		accountManager:             accountManager,
		bartItemManager:            bartItemManager,
		rateLimitClasses:           classes,
		timeNow:                    time.Now,
		maxConcurrentLoginsPerUser: MaxConcurrentLoginsPerUser,
		createAccount:              createAccount,
		logger:                     logger,
	}
}

// RegisterChatSession adds a user to a chat room. The authCookie param is an
// opaque token returned by {{OServiceService.ServiceRequest}} that identifies the user and chat room.
// It returns the session object registered in the ChatSessionRegistry.
// This method does not verify that the user and chat room exist because it
// implicitly trusts the contents of the token signed by {{OServiceService.ServiceRequest}}.
func (s AuthService) RegisterChatSession(ctx context.Context, serverCookie state.ServerCookie) (*state.SessionInstance, error) {
	sess, err := s.chatSessionRegistry.AddSession(ctx, serverCookie.ChatCookie, serverCookie.ScreenName)
	if err != nil {
		return nil, errors.New("AddSession: " + err.Error())
	}

	sess.Session().SetRateClasses(time.Now(), s.rateLimitClasses)
	return sess, err
}

// RegisterBOSSession adds a new session to the session registry.
func (s AuthService) RegisterBOSSession(ctx context.Context, serverCookie state.ServerCookie) (*state.SessionInstance, error) {
	u, err := s.userManager.User(ctx, serverCookie.ScreenName.IdentScreenName())
	if err != nil {
		return nil, errors.New("failed to retrieve user: " + err.Error())
	} else if u == nil {
		return nil, errors.New("user not found")
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var doMultiSess bool
	flag := wire.MultiConnFlag(serverCookie.MultiConnFlag)
	if flag == wire.MultiConnFlagsRecentClient {
		doMultiSess = true
	}

	sess, err := s.sessionManager.AddSession(ctx, u.DisplayScreenName, doMultiSess)
	if err != nil {
		return nil, errors.New("AddSession: " + err.Error())
	}

	// set the unconfirmed user info flag if this account is unconfirmed
	if confirmed, err := s.accountManager.ConfirmStatus(ctx, sess.IdentScreenName()); err != nil {
		return nil, errors.New("error setting unconfirmed user flag: " + err.Error())
	} else if !confirmed {
		sess.SetUserInfoFlag(wire.OServiceUserFlagUnconfirmed)
	}

	if u.IsBot {
		sess.SetUserInfoFlag(wire.OServiceUserFlagBot)
	}

	sess.SetKerberosAuth(serverCookie.KerberosAuth == 1)
	sess.Session().SetSignonTime(time.Now())
	sess.Session().SetRateClasses(time.Now(), s.rateLimitClasses)
	// set string containing OSCAR client name and version
	sess.SetClientID(serverCookie.ClientID)
	sess.Session().SetMemberSince(time.Now())
	sess.Session().SetOfflineMsgCount(u.OfflineMsgCount)
	if _, alreadySet := sess.Session().BuddyIcon(); !alreadySet {
		if bartID, err := s.bartItemManager.BuddyIconMetadata(ctx, sess.IdentScreenName()); err != nil {
			return nil, errors.New("BuddyIconMetadata: " + err.Error())
		} else if bartID != nil {
			sess.Session().SetBuddyIcon(*bartID)
		}
	}

	// indicate whether the client supports/wants multiple concurrent sessions
	sess.SetMultiConnFlag(flag)
	if u.DisplayScreenName.IsUIN() {
		sess.SetUserInfoFlag(wire.OServiceUserFlagICQ)
		uin, err := strconv.Atoi(u.IdentScreenName.String())
		if err != nil {
			return nil, errors.New("error converting username to UIN: " + err.Error())
		}

		sess.Session().SetUIN(uint32(uin))
	}

	return sess, nil
}

// RetrieveBOSSession returns a user's existing session instance.
func (s AuthService) RetrieveBOSSession(ctx context.Context, serverCookie state.ServerCookie) (*state.SessionInstance, error) {
	u, err := s.userManager.User(ctx, serverCookie.ScreenName.IdentScreenName())
	if err != nil {
		return nil, errors.New("failed to retrieve user: " + err.Error())
	} else if u == nil {
		return nil, errors.New("user not found")
	}

	sess := s.sessionRetriever.RetrieveSession(u.IdentScreenName)
	if sess == nil {
		return nil, nil
	}

	return sess.Instance(serverCookie.SessionNum), nil
}

func (s AuthService) CrackCookie(authCookie []byte) (state.ServerCookie, error) {
	c := state.ServerCookie{}
	buf, err := s.cookieBaker.Crack(authCookie)
	if err != nil {
		return c, err
	}

	if err := wire.UnmarshalBE(&c, bytes.NewBuffer(buf)); err != nil {
		return c, err
	}

	return c, nil
}

func (s AuthService) loginSuccessResponse(props loginProperties, advertisedHost string) (wire.TLVRestBlock, error) {
	loginCookie := state.ServerCookie{
		Service:       wire.BOS,
		ScreenName:    props.screenName,
		ClientID:      props.clientID,
		MultiConnFlag: props.multiConnFlag,
	}
	if props.isKerberosPlaintextAuth || props.isKerberosRoastedAuth {
		loginCookie.KerberosAuth = 1
	}

	buf := &bytes.Buffer{}
	if err := wire.MarshalBE(loginCookie, buf); err != nil {
		return wire.TLVRestBlock{}, err
	}

	cookie, err := s.cookieBaker.Issue(buf.Bytes())
	if err != nil {
		return wire.TLVRestBlock{}, errors.New("failed to issue auth cookie: " + err.Error())
	}

	reconnectHost := advertisedHost
	sslState := wire.OServiceServiceResponseSSLStateNotUsed
	s.logger.Debug("loginSuccessResponse: returning login response", "screen_name", props.screenName, "reconnect_host", reconnectHost, "ssl_state", sslState)
	return wire.TLVRestBlock{
		TLVList: []wire.TLV{
			wire.NewTLVBE(wire.LoginTLVTagsScreenName, props.screenName),
			wire.NewTLVBE(wire.LoginTLVTagsReconnectHere, reconnectHost),
			wire.NewTLVBE(wire.LoginTLVTagsAuthorizationCookie, cookie),
			wire.NewTLVBE(wire.OServiceTLVTagsSSLState, sslState),
		},
	}, nil
}

// loginProperties represents the properties sent by the client at login.
type loginProperties struct {
	clientID                string
	isBUCPAuth              bool
	isFLAPAuth              bool
	isFLAPJavaAuth          bool
	isKerberosPlaintextAuth bool
	isKerberosRoastedAuth   bool
	isTOCAuth               bool
	multiConnFlag           uint8
	passwordHash            []byte
	plaintextPassword       []byte
	roastedPass             []byte
	screenName              state.DisplayScreenName
}

// fromTLV creates an instance of loginProperties from a TLV list.
func (l *loginProperties) fromTLV(list wire.TLVList) error {
	// extract screen name
	if screenName, found := list.String(wire.LoginTLVTagsScreenName); found {
		l.screenName = state.DisplayScreenName(screenName)
	} else {
		return errors.New("screen name doesn't exist in tlv")
	}

	// extract client name and version
	if clientID, found := list.String(wire.LoginTLVTagsClientIdentity); found {
		l.clientID = clientID
	}

	// get the password from the appropriate TLV
	// older clients have a roasted password,
	// newer clients have a hashed password
	// ICQ may omit the password TLV when logging in without saved password
	switch {
	case list.HasTag(wire.LoginTLVTagsPasswordHash):
		// extract password hash for BUCP login
		l.passwordHash, _ = list.Bytes(wire.LoginTLVTagsPasswordHash)
		l.isBUCPAuth = true
	case list.HasTag(wire.LoginTLVTagsRoastedPassword):
		// extract roasted password for FLAP login
		l.roastedPass, _ = list.Bytes(wire.LoginTLVTagsRoastedPassword)
		if strings.HasPrefix(l.clientID, "AOL Instant Messenger (TM) version") &&
			strings.Contains(l.clientID, "for Java") {
			l.isFLAPJavaAuth = true
		} else {
			l.isFLAPAuth = true
		}
	case list.HasTag(wire.LoginTLVTagsRoastedTOCPassword):
		// extract roasted password for TOC FLAP login
		l.roastedPass, _ = list.Bytes(wire.LoginTLVTagsRoastedTOCPassword)
		l.isTOCAuth = true
	case list.HasTag(wire.LoginTLVTagsPlaintextPassword):
		l.plaintextPassword, _ = list.Bytes(wire.LoginTLVTagsPlaintextPassword)
		l.isKerberosPlaintextAuth = true
	case list.HasTag(wire.LoginTLVTagsRoastedKerberosPassword):
		l.roastedPass, _ = list.Bytes(wire.LoginTLVTagsRoastedKerberosPassword)
		l.isKerberosRoastedAuth = true
	default:
		l.isFLAPAuth = true
	}

	// does the client support multiple concurrent sessions?
	if multiConnFlags, found := list.Uint8(wire.LoginTLVTagsMultiConnFlags); found {
		l.multiConnFlag = multiConnFlags
	}

	return nil
}

func loginFailureResponse(props loginProperties, errCode uint16) wire.TLVRestBlock {
	return wire.TLVRestBlock{
		TLVList: []wire.TLV{
			wire.NewTLVBE(wire.LoginTLVTagsScreenName, props.screenName),
			wire.NewTLVBE(wire.LoginTLVTagsErrorSubcode, errCode),
		},
	}
}
