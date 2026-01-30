package handlers

// CookieBaker issues and validates authentication cookies for OSCAR services.
type CookieBaker interface {
	// Issue creates a new authentication cookie from the given payload.
	Issue(data []byte) ([]byte, error)
	// Crack verifies and decodes an authentication cookie.
	Crack(data []byte) ([]byte, error)
}
