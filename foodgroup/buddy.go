package foodgroup

// buddyNotifier centralizes logic for sending buddy arrival and departure notifications.
type buddyNotifier struct {
	bartItemManager     BARTItemManager
	relationshipFetcher RelationshipFetcher
	messageRelayer      MessageRelayer
	sessionRetriever    SessionRetriever
}
