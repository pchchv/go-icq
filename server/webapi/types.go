package webapi

// OSCARConfig provides configuration for OSCAR services.
type OSCARConfig interface {
	GetSSLBOSAddress() (host string, port int)
	GetBOSAddress() (host string, port int)
	IsSSLAvailable() bool
	IsAuthDisabled() bool
}
