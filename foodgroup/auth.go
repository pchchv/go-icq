package foodgroup

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
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
		return nil, fmt.Errorf("AddSession: %w", err)
	}

	sess.Session().SetRateClasses(time.Now(), s.rateLimitClasses)
	return sess, err
}

// RegisterBOSSession adds a new session to the session registry.
func (s AuthService) RegisterBOSSession(ctx context.Context, serverCookie state.ServerCookie) (*state.SessionInstance, error) {
	u, err := s.userManager.User(ctx, serverCookie.ScreenName.IdentScreenName())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user: %w", err)
	} else if u == nil {
		return nil, fmt.Errorf("user not found")
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
		return nil, fmt.Errorf("AddSession: %w", err)
	}

	// set the unconfirmed user info flag if this account is unconfirmed
	if confirmed, err := s.accountManager.ConfirmStatus(ctx, sess.IdentScreenName()); err != nil {
		return nil, fmt.Errorf("error setting unconfirmed user flag: %w", err)
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
			return nil, fmt.Errorf("BuddyIconMetadata: %w", err)
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
			return nil, fmt.Errorf("error converting username to UIN: %w", err)
		}

		sess.Session().SetUIN(uint32(uin))
	}

	return sess, nil
}
