package handlers

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
