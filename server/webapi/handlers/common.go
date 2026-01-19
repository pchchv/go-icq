package handlers

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/pchchv/go-icq/state"
)

// CommonHandler provides shared utilities for all Web API handlers.
type CommonHandler struct {
	Logger *slog.Logger
}

// SessionRetriever provides methods to retrieve OSCAR sessions.
type SessionRetriever interface {
	AllSessions() []*state.Session
	RetrieveSession(screenName state.IdentScreenName) *state.Session
}

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

// ResponseBody contains the status and data for API responses.
type ResponseBody struct {
	StatusCode int         `json:"statusCode" xml:"statusCode"`
	StatusText string      `json:"statusText" xml:"statusText"`
	Data       interface{} `json:"data,omitempty" xml:"data,omitempty"`
}

// BaseResponse is the standard response envelope for all Web API responses.
// It supports both JSON and XML marshaling.
type BaseResponse struct {
	XMLName  xml.Name     `xml:"response" json:"-"`
	Response ResponseBody `json:"response"`
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

// SendXML sends an XML response.
func SendXML(w http.ResponseWriter, data interface{}, logger *slog.Logger) {
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	// convert BaseResponse with map data to a format XML can handle
	if baseResp, ok := data.(BaseResponse); ok {
		data = convertBaseResponseForXML(baseResp)
	}

	// marshal the data
	xmlData, err := xml.Marshal(data)
	if err != nil {
		if logger != nil {
			logger.Error("failed to marshal XML response", "err", err.Error())
		}

		SendXMLError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// write XML declaration and data
	xmlOutput := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>%s`, xmlData)
	// set content length for proper response handling
	w.Header().Set("Content-Length", strconv.Itoa(len(xmlOutput)))
	w.Write([]byte(xmlOutput))
}

// SendXMLError sends an XML error response.
func SendXMLError(w http.ResponseWriter, statusCode int, message string) {
	resp := ErrorResponse{}
	resp.StatusCode = statusCode
	resp.StatusText = message

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.WriteHeader(statusCode)

	// write XML declaration and marshal the response
	xmlData, err := xml.Marshal(resp)
	if err != nil {
		// fall back to simple text response
		http.Error(w, message, statusCode)
		return
	}

	xmlOutput := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>%s`, xmlData)
	w.Write([]byte(xmlOutput))
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

// convertBaseResponseForXML converts a BaseResponse with map data to XMLMapResponse.
func convertBaseResponseForXML(resp BaseResponse) XMLMapResponse {
	xmlResp := XMLMapResponse{
		StatusCode: resp.Response.StatusCode,
		StatusText: resp.Response.StatusText,
	}

	// convert map data to XMLData struct
	if dataMap, ok := resp.Response.Data.(map[string]interface{}); ok {
		xmlData := XMLData{}
		// handle auth response fields
		if tokenData, ok := dataMap["token"].(map[string]interface{}); ok {
			xmlData.Token = &XMLToken{}
			if a, ok := tokenData["a"].(string); ok {
				xmlData.Token.A = a
			}

			if expiresIn, ok := tokenData["expiresIn"].(int); ok {
				xmlData.Token.ExpiresIn = expiresIn
			}
		}

		if loginId, ok := dataMap["loginId"].(string); ok {
			xmlData.LoginID = loginId
		}

		if screenName, ok := dataMap["screenName"].(string); ok {
			xmlData.ScreenName = screenName
		}

		if sessionSecret, ok := dataMap["sessionSecret"].(string); ok {
			xmlData.SessionSecret = sessionSecret
		}

		if hostTime, ok := dataMap["hostTime"].(int64); ok {
			xmlData.HostTime = hostTime
		}

		if tokenExpiresIn, ok := dataMap["tokenExpiresIn"].(int); ok {
			xmlData.TokenExpiresIn = tokenExpiresIn
		}

		// handle session response fields
		if aimsid, ok := dataMap["aimsid"].(string); ok {
			xmlData.AimSID = aimsid
		}

		if fetchUrl, ok := dataMap["fetchUrl"].(string); ok {
			xmlData.FetchURL = fetchUrl
		}

		// handle message response fields
		if msgId, ok := dataMap["msgId"].(string); ok {
			xmlData.MsgID = msgId
		}

		if state, ok := dataMap["state"].(string); ok {
			xmlData.State = state
		}

		xmlResp.Data = xmlData
	}

	return xmlResp
}
