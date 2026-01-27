package handlers

import "github.com/pchchv/go-icq/server/webapi/types"

// FetchEventsData contains the events and metadata.
type FetchEventsData struct {
	Events          []types.Event `json:"events"`
	LastSeqNum      uint64        `json:"lastSeqNum"`
	TimeToNextFetch int           `json:"timeToNextFetch"`
	FetchBaseURL    string        `json:"fetchBaseURL"`
}
