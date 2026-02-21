package foodgroup

// PermitDenyService provides functionality for the PermitDeny (PD) food group.
// The PD food group manages settings for permit/deny (allow/block)
// for pre-feedbag (sever-side buddy list) AIM clients.
type PermitDenyService struct {
	buddyBroadcaster           buddyBroadcaster
	clientSideBuddyListManager ClientSideBuddyListManager
}

// NewPermitDenyService creates an instance of PermitDenyService.
func NewPermitDenyService(
	bartItemManager BARTItemManager,
	relationshipFetcher RelationshipFetcher,
	clientSideBuddyListManager ClientSideBuddyListManager,
	messageRelayer MessageRelayer,
	sessionRetriever SessionRetriever,
) PermitDenyService {
	return PermitDenyService{
		buddyBroadcaster:           newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		clientSideBuddyListManager: clientSideBuddyListManager,
	}
}
