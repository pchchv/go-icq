package foodgroup

import (
	"log/slog"
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
