package handlers

import "github.com/pchchv/go-icq/server/webapi/types"

// FetchEventsData contains the events and metadata.
type FetchEventsData struct {
	Events          []types.Event `json:"events"`
	LastSeqNum      uint64        `json:"lastSeqNum"`
	TimeToNextFetch int           `json:"timeToNextFetch"`
	FetchBaseURL    string        `json:"fetchBaseURL"`
}

// FetchEventsResponse represents the response for fetchEvents endpoint.
type FetchEventsResponse struct {
	Response struct {
		StatusCode int             `json:"statusCode"`
		StatusText string          `json:"statusText"`
		Data       FetchEventsData `json:"data"`
	} `json:"response"`
}
