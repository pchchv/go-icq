package foodgroup

import "log/slog"

type BARTService struct {
	bartItemManager        BARTItemManager
	buddyUpdateBroadcaster buddyBroadcaster
	messageRelayer         MessageRelayer
	logger                 *slog.Logger
}

func NewBARTService(
	logger *slog.Logger,
	bartItemManager BARTItemManager,
	messageRelayer MessageRelayer,
	relationshipFetcher RelationshipFetcher,
	sessionRetriever SessionRetriever,
) BARTService {
	return BARTService{
		bartItemManager:        bartItemManager,
		buddyUpdateBroadcaster: newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		messageRelayer:         messageRelayer,
		logger:                 logger,
	}
}
