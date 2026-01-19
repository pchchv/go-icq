package handlers

import "encoding/xml"

// XMLToken represents the token structure in XML.
type XMLToken struct {
	A         string `xml:"a"`
	ExpiresIn int    `xml:"expiresIn"`
}

// XMLData wraps the data for XML responses.
type XMLData struct {
	// Auth response fields
	Token          *XMLToken `xml:"token,omitempty"`
	LoginID        string    `xml:"loginId,omitempty"`
	ScreenName     string    `xml:"screenName,omitempty"`
	SessionSecret  string    `xml:"sessionSecret,omitempty"`
	HostTime       int64     `xml:"hostTime,omitempty"`
	TokenExpiresIn int       `xml:"tokenExpiresIn,omitempty"`
	// Generic fields for other responses
	AimSID   string `xml:"aimsid,omitempty"`
	FetchURL string `xml:"fetchUrl,omitempty"`
	MsgID    string `xml:"msgId,omitempty"`
	State    string `xml:"state,omitempty"`
	// For any other data, we'll encode as string
	Raw string `xml:",chardata"`
}

// ErrorResponse represents an error response with proper XML/JSON support.
type ErrorResponse struct {
	XMLName  xml.Name `xml:"response" json:"-"`
	Response struct {
		StatusCode int    `json:"statusCode" xml:"statusCode"`
		StatusText string `json:"statusText" xml:"statusText"`
	} `json:"response" xml:"-"`
	// For XML responses, flatten the structure
	StatusCode int    `json:"-" xml:"statusCode"`
	StatusText string `json:"-" xml:"statusText"`
}

// XMLMapResponse is a helper struct for converting map-based responses to XML.
type XMLMapResponse struct {
	XMLName    xml.Name `xml:"response"`
	StatusCode int      `xml:"statusCode"`
	StatusText string   `xml:"statusText"`
	Data       XMLData  `xml:"data,omitempty"`
}
