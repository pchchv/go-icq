package webapi

import (
	"strings"

	"github.com/pchchv/go-icq/config"
)

// OSCARConfigAdapter adapts the main server configuration to
// provide OSCAR-specific configuration for the Web API bridge.
type OSCARConfigAdapter struct {
	cfg       config.Config
	listeners []config.Listener
}

// NewOSCARConfigAdapter creates a new OSCAR configuration adapter.
func NewOSCARConfigAdapter(cfg config.Config) *OSCARConfigAdapter {
	listeners, _ := cfg.ParseListenersCfg()
	return &OSCARConfigAdapter{
		cfg:       cfg,
		listeners: listeners,
	}
}

// IsSSLAvailable checks if any listener has SSL configured.
func (a *OSCARConfigAdapter) IsSSLAvailable() bool {
	for _, listener := range a.listeners {
		if listener.HasSSL {
			return true
		}
	}
	return false
}

// IsAuthDisabled returns whether authentication is disabled.
func (a *OSCARConfigAdapter) IsAuthDisabled() bool {
	return a.cfg.DisableAuth
}

// splitHostPort splits a host:port string, handling IPv6 addresses correctly.
// Unlike net.SplitHostPort, this doesn't return an error for missing ports.
func splitHostPort(hostport string) (host string, port string) {
	// handle IPv6 addresses
	if strings.HasPrefix(hostport, "[") {
		if endIdx := strings.LastIndex(hostport, "]"); endIdx != -1 {
			if host = hostport[1:endIdx]; endIdx+1 < len(hostport) && hostport[endIdx+1] == ':' {
				port = hostport[endIdx+2:]
			}
			return
		}
	}

	// handle IPv4 and hostnames
	if lastColon := strings.LastIndex(hostport, ":"); lastColon != -1 {
		// check if this might be an IPv6 address without brackets
		if strings.Count(hostport, ":") > 1 {
			// multiple colons, likely IPv6 without port
			host = hostport
			return
		}

		host = hostport[:lastColon]
		port = hostport[lastColon+1:]
		return
	}

	// no port specified
	host = hostport
	return
}
