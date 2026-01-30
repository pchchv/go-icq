package handlers

import (
	"context"
	"encoding/hex"
	"encoding/xml"
	"log/slog"
	"net/http"

	"github.com/pchchv/go-icq/state"
)

// CookieBaker issues and validates authentication cookies for OSCAR services.
type CookieBaker interface {
	// Issue creates a new authentication cookie from the given payload.
	Issue(data []byte) ([]byte, error)
	// Crack verifies and decodes an authentication cookie.
	Crack(data []byte) ([]byte, error)
}

// StartOSCARSessionRequest represents the request parameters for startOSCARSession.
type StartOSCARSessionRequest struct {
	AimSID   string // WebAPI session ID
	UseSSL   bool   // Whether to use SSL for the OSCAR connection
	Compress bool   // Whether to use compression (not implemented)
}

// StartOSCARSessionResponse represents the response for startOSCARSession endpoint.
type StartOSCARSessionResponse struct {
	XMLName  xml.Name `xml:"response" json:"-"`
	Response struct {
		StatusCode int    `json:"statusCode" xml:"statusCode"`
		StatusText string `json:"statusText" xml:"statusText"`
		Data       struct {
			Host        string `json:"host" xml:"host"`
			Port        int    `json:"port" xml:"port"`
			Cookie      string `json:"cookie" xml:"cookie"`
			UseSSL      bool   `json:"useSSL" xml:"useSSL"`
			Encryption  string `json:"encryption,omitempty" xml:"encryption,omitempty"`
			Compression string `json:"compression,omitempty" xml:"compression,omitempty"`
		} `json:"data" xml:"data"`
	} `json:"response" xml:"-"`
	// For XML responses, flatten the structure.
	StatusCode int    `json:"-" xml:"statusCode"`
	StatusText string `json:"-" xml:"statusText"`
	Data       struct {
		Host        string `json:"-" xml:"host"`
		Port        int    `json:"-" xml:"port"`
		Cookie      string `json:"-" xml:"cookie"`
		UseSSL      bool   `json:"-" xml:"useSSL"`
		Encryption  string `json:"-" xml:"encryption,omitempty"`
		Compression string `json:"-" xml:"compression,omitempty"`
	} `json:"-" xml:"data"`
}

// OSCARConfig provides configuration for OSCAR services.
type OSCARConfig interface {
	// GetBOSAddress returns the BOS server address for client connections.
	GetBOSAddress() (host string, port int)
	// GetSSLBOSAddress returns the SSL-enabled BOS server address.
	GetSSLBOSAddress() (host string, port int)
	// IsSSLAvailable checks if SSL is configured for BOS connections.
	IsSSLAvailable() bool
	// IsAuthDisabled returns whether authentication is disabled.
	IsAuthDisabled() bool
}

// OSCARBridgeStore manages the persistence of OSCAR bridge sessions.
type OSCARBridgeStore interface {
	// SaveBridgeSession stores the mapping between WebAPI and OSCAR sessions.
	SaveBridgeSession(ctx context.Context, webSessionID string, oscarCookie []byte, bosHost string, bosPort int) error
	// GetBridgeSession retrieves bridge session details.
	GetBridgeSession(ctx context.Context, webSessionID string) (*state.OSCARBridgeSession, error)
	// DeleteBridgeSession removes a bridge session.
	DeleteBridgeSession(ctx context.Context, webSessionID string) error
}

// OSCARAuthService defines methods needed for OSCAR authentication and session management.
type OSCARAuthService interface {
	// RegisterBOSSession creates a new BOS (Basic OSCAR Service) sessio0n.
	RegisterBOSSession(ctx context.Context, authCookie state.ServerCookie) (*state.SessionInstance, error)
	// RetrieveBOSSession retrieves an existing BOS session.
	RetrieveBOSSession(ctx context.Context, authCookie state.ServerCookie) (*state.SessionInstance, error)
	// Signout ends an OSCAR session.
	Signout(ctx context.Context, instance *state.SessionInstance)
}

// OSCARBridgeHandler handles Web API to OSCAR protocol bridging endpoints.
// This handler is responsible for creating a
// bridge between web-based clients and the native OSCAR protocol,
// allowing web clients to connect to OSCAR services.
type OSCARBridgeHandler struct {
	OSCARAuthService OSCARAuthService
	SessionManager   *state.WebAPISessionManager
	CookieBaker      CookieBaker
	BridgeStore      OSCARBridgeStore
	Config           OSCARConfig
	Logger           *slog.Logger
}

// hasOSCARBridgeCapability checks if the API key has permission to create OSCAR bridges.
func (h *OSCARBridgeHandler) hasOSCARBridgeCapability(apiKey *state.WebAPIKey) bool {
	if len(apiKey.Capabilities) == 0 {
		// no restrictions if capabilities not specified
		return true
	}

	// check if OSCAR bridge is explicitly enabled
	for _, cap := range apiKey.Capabilities {
		if cap == "oscar_bridge" || cap == "*" {
			return true
		}
	}

	return false
}

// sendError sends an error response in the appropriate format.
func (h *OSCARBridgeHandler) sendError(w http.ResponseWriter, _ *http.Request, statusCode int, message string) {
	// SendError already detects format from Content-Type header.
	SendError(w, statusCode, message)
}

// buildResponse constructs the response object.
func (h *OSCARBridgeHandler) buildResponse(host string, port int, cookie []byte, useSSL, compress bool) *StartOSCARSessionResponse {
	resp := &StartOSCARSessionResponse{}
	resp.Response.StatusCode = 200
	resp.Response.StatusText = "OK"
	resp.Response.Data.Host = host
	resp.Response.Data.Port = port
	resp.Response.Data.Cookie = hex.EncodeToString(cookie) // hex encode the cookie
	resp.Response.Data.UseSSL = useSSL

	// add encryption info if SSL is used
	if useSSL {
		resp.Response.Data.Encryption = "TLS"
	}

	// add compression info if requested (not implemented)
	if compress {
		resp.Response.Data.Compression = "none" // Compression not implemented
	}

	// duplicate data for XML format
	resp.StatusCode = resp.Response.StatusCode
	resp.StatusText = resp.Response.StatusText
	resp.Data.Host = resp.Response.Data.Host
	resp.Data.Port = resp.Response.Data.Port
	resp.Data.Cookie = resp.Response.Data.Cookie
	resp.Data.UseSSL = resp.Response.Data.UseSSL
	resp.Data.Encryption = resp.Response.Data.Encryption
	resp.Data.Compression = resp.Response.Data.Compression
	return resp
}

// sendResponse sends the response in the requested format.
func (h *OSCARBridgeHandler) sendResponse(w http.ResponseWriter, r *http.Request, resp *StartOSCARSessionResponse) {
	// Use the centralized SendResponse function which handles all formats.
	SendResponse(w, r, resp, h.Logger)
}
