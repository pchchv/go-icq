package foodgroup

import (
	"log/slog"
	"time"

	"github.com/pchchv/go-icq/config"
	"github.com/pchchv/go-icq/wire"
)

// OServiceService provides functionality for the OService food group,
// which provides an assortment of services useful across multiple food groups.
type OServiceService struct {
	buddyBroadcaster      buddyBroadcaster
	cfg                   config.Config // todo remove
	logger                *slog.Logger
	snacRateLimits        wire.SNACRateLimits
	timeNow               func() time.Time
	chatRoomManager       ChatRoomRegistry
	cookieIssuer          CookieBaker
	messageRelayer        MessageRelayer
	chatMessageRelayer    ChatMessageRelayer
	profileManager        ProfileManager
	offlineMessageManager OfflineMessageManager
}

// NewOServiceService creates a new instance of NewOServiceService.
func NewOServiceService(
	cfg config.Config,
	messageRelayer MessageRelayer,
	logger *slog.Logger,
	cookieIssuer CookieBaker,
	chatRoomManager ChatRoomRegistry,
	relationshipFetcher RelationshipFetcher,
	sessionRetriever SessionRetriever,
	bartItemManager BARTItemManager,
	snacRateLimits wire.SNACRateLimits,
	chatMessageRelayer ChatMessageRelayer,
	profileManager ProfileManager,
	offlineMessageManager OfflineMessageManager,
) *OServiceService {
	return &OServiceService{
		cookieIssuer:          cookieIssuer,
		messageRelayer:        messageRelayer,
		buddyBroadcaster:      newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		cfg:                   cfg,
		logger:                logger,
		snacRateLimits:        snacRateLimits,
		timeNow:               time.Now,
		chatRoomManager:       chatRoomManager,
		chatMessageRelayer:    chatMessageRelayer,
		profileManager:        profileManager,
		offlineMessageManager: offlineMessageManager,
	}
}
