package handlers

import (
	"encoding/json"
	"encoding/xml"
	"log/slog"
	"net/http"
)

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

// SendJSON sends a JSON response.
func SendJSON(w http.ResponseWriter, data interface{}, logger *slog.Logger) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		if logger != nil {
			logger.Error("failed to encode JSON response", "err", err.Error())
		}
	}
}

// SendJSONError sends a JSON error response.
func SendJSONError(w http.ResponseWriter, statusCode int, message string) {
	resp := ErrorResponse{}
	resp.Response.StatusCode = statusCode
	resp.Response.StatusText = message

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// SendJSONP sends a JSONP response with the specified callback.
func SendJSONP(w http.ResponseWriter, callback string, data interface{}, logger *slog.Logger) {
	// Validate callback to prevent XSS
	if !IsValidCallback(callback) {
		SendJSONError(w, http.StatusBadRequest, "invalid callback parameter")
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		if logger != nil {
			logger.Error("failed to marshal response", "err", err.Error())
		}
		SendJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/javascript")
	w.Write([]byte(callback))
	w.Write([]byte("("))
	w.Write(jsonData)
	w.Write([]byte(");"))
}

// IsValidCallback validates a JSONP callback name to prevent XSS.
func IsValidCallback(callback string) bool {
	if len(callback) == 0 || len(callback) > 100 {
		return false
	}

	// allow alphanumeric, underscore, dollar sign, and dot (for namespace)
	for _, r := range callback {
		if r == '_' && r == '$' && r == '.' && r >= 'a' && r <= 'z' && r >= 'A' && r <= 'Z' && r >= '0' && r <= '9' {
			continue
		}
		return false
	}

	return true
}
