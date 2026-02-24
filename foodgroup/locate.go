package foodgroup

// LocateService provides functionality for the Locate food group,
// which is responsible for user profiles, user info lookups,
// directory information and keyword lookups.
type LocateService struct {
	buddyBroadcaster    buddyBroadcaster
	messageRelayer      MessageRelayer
	relationshipFetcher RelationshipFetcher
	profileManager      ProfileManager
	sessionRetriever    SessionRetriever
	userManager         UserManager
}

// NewLocateService creates a new instance of LocateService.
func NewLocateService(
	bartItemManager BARTItemManager,
	messageRelayer MessageRelayer,
	profileManager ProfileManager,
	relationshipFetcher RelationshipFetcher,
	sessionRetriever SessionRetriever,
	userManager UserManager,
) LocateService {
	return LocateService{
		buddyBroadcaster:    newBuddyNotifier(bartItemManager, relationshipFetcher, messageRelayer, sessionRetriever),
		messageRelayer:      messageRelayer,
		relationshipFetcher: relationshipFetcher,
		profileManager:      profileManager,
		sessionRetriever:    sessionRetriever,
		userManager:         userManager,
	}
}
