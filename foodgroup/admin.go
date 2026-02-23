package foodgroup

import "log/slog"

// AdminService provides functionality for the Admin food group.
// The Admin food group is used for client control of passwords,
// screen name formatting, email address, and account confirmation.
type AdminService struct {
	accountManager   AccountManager
	buddyBroadcaster buddyBroadcaster
	messageRelayer   MessageRelayer
	logger           *slog.Logger
}

// NewAdminService creates an instance of AdminService.
func NewAdminService(
	accountManager AccountManager,
	bartItemManager BARTItemManager,
	relationshipFetcher RelationshipFetcher,
	messageRelayer MessageRelayer,
	sessionRetriever SessionRetriever,
	logger *slog.Logger,
) *AdminService {
	return &AdminService{
		accountManager:   accountManager,
		buddyBroadcaster: newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		messageRelayer:   messageRelayer,
		logger:           logger,
	}
}
