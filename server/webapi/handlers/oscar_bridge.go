package handlers

import "encoding/xml"

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
