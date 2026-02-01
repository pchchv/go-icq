package webapi

import (
	"net"
	"strconv"
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

// GetSSLBOSAddress returns the SSL-enabled BOS server address for client connections.
func (a *OSCARConfigAdapter) GetSSLBOSAddress() (host string, port int) {
	// default to first listener configuration with SSL
	for _, listener := range a.listeners {
		if listener.HasSSL && listener.BOSAdvertisedHostSSL != "" {
			host, portStr := splitHostPort(listener.BOSAdvertisedHostSSL)
			if portStr != "" {
				if p, err := strconv.Atoi(portStr); err == nil {
					port = p
				}
			}

			if port == 0 {
				port = 5190 // default OSCAR SSL port (could be different)
			}

			return host, port
		}
	}

	// fall back to plain address if no SSL configured
	return a.GetBOSAddress()
}

// GetBOSAddress returns the plain (non-SSL) BOS server address for client connections.
// This parses the configured BOS advertised host to extract the hostname and port.
func (a *OSCARConfigAdapter) GetBOSAddress() (host string, port int) {
	// default to first listener configuration
	if len(a.listeners) == 0 {
		return "localhost", 5190 // default OSCAR port
	}

	listener := a.listeners[0]
	// parse the advertised host for plain connections
	if listener.BOSAdvertisedHostPlain != "" {
		host, portStr := splitHostPort(listener.BOSAdvertisedHostPlain)
		if portStr != "" {
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		if port == 0 {
			port = 5190 // default OSCAR port
		}

		return host, port
	}

	// fall back to parsing the listen address
	if listener.BOSListenAddress != "" {
		host, portStr, err := net.SplitHostPort(listener.BOSListenAddress)
		if err == nil {
			if host == "" {
				host = "localhost"
			}

			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
			}
		}

		if port == 0 {
			port = 5190
		}
		
		return host, port
	}

	return "localhost", 5190
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
