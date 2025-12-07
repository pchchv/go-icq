package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var (
	// Simple error for duplicate listener definitions.
	errDuplicateListener = errors.New("duplicate listener definition")
	// Simple error for missing BOS listeners.
	errNoBOSListeners = errors.New("at least one BOS listener is required")
)

type Listener struct {
	BOSListenAddress       string
	BOSAdvertisedHostPlain string
	BOSAdvertisedHostSSL   string
	KerberosListenAddress  string
	HasSSL                 bool
}

type Build struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

//go:generate go run ../cmd/config_generator unix settings.env ssl
type Config struct {
	BOSListeners            []string `envconfig:"GO_ICQ_LISTENERS" required:"true" basic:"LOCAL://0.0.0.0:5190" ssl:"LOCAL://0.0.0.0:5190" description:"Network listeners for core GO-ICQ services. For multi-homed servers, allows users to connect from multiple networks. For example, you can allow both LAN and Internet clients to connect to the same server using different connection settings.\n\nFormat:\n\t- Comma-separated list of [NAME]://[HOSTNAME]:[PORT]\n\t- Listener names and ports must be unique\n\t- Listener names are user-defined\n\t- Each listener needs a listener in GO_ICQ_ADVERTISED_LISTENERS_PLAIN\n\nExamples:\n\t// Listen on all interfaces\n\tLAN://0.0.0.0:5190\n\t// Separate Internet and LAN config\n\tWAN://142.250.176.206:5190,LAN://192.168.1.10:5191"`
	BOSAdvertisedHostsPlain []string `envconfig:"GO_ICQ_ADVERTISED_LISTENERS_PLAIN" required:"true" basic:"LOCAL://127.0.0.1:5190" ssl:"LOCAL://127.0.0.1:5190" description:"Hostnames published by the server that clients connect to for accessing various GO-ICQ services. These hostnames are NOT the bind addresses. For multi-homed use servers, allows clients to connect using separate hostnames per network.\n\nFormat:\n\t- Comma-separated list of [NAME]://[HOSTNAME]:[PORT]\n\t- Each listener config must correspond to a config in GO_ICQ_LISTENERS\n\t- Clients MUST be able to connect to these hostnames\n\nExamples:\n\t// Local LAN config, server behind NAT\n\tLAN://192.168.1.10:5190\n\t// Separate Internet and LAN config\n\tWAN://aim.example.com:5190,LAN://192.168.1.10:5191"`
	BOSAdvertisedHostsSSL   []string `envconfig:"GO_ICQ_ADVERTISED_LISTENERS_SSL" required:"false" basic:"" ssl:"LOCAL://ras.dev:5193" description:"Same as GO_ICQ_ADVERTISED_LISTENERS_PLAIN, except the hostname is for the server that terminates SSL."`
	KerberosListeners       []string `envconfig:"KERBEROS_LISTENERS" required:"false" basic:"" ssl:"LOCAL://0.0.0.0:1088" description:"Network listeners for Kerberos authentication. See GO_ICQ_LISTENERS doc for more details.\n\nExamples:\n\t// Listen on all interfaces\n\tLAN://0.0.0.0:1088\n\t// Separate Internet and LAN config\n\tWAN://142.250.176.206:1088,LAN://192.168.1.10:1087"`
	TOCListeners            []string `envconfig:"TOC_LISTENERS" required:"true" basic:"0.0.0.0:9898" ssl:"0.0.0.0:9898" description:"Network listeners for TOC protocol service.\n\nFormat: Comma-separated list of hostname:port pairs.\n\nExamples:\n\t// All interfaces\n\t0.0.0.0:9898\n\t// Multiple listeners\n\t0.0.0.0:9898,192.168.1.10:9899"`
	DisableAuth             bool     `envconfig:"DISABLE_AUTH" required:"true" basic:"true" ssl:"true" description:"Disable password check and auto-create new users at login time. Useful for quickly creating new accounts during development without having to register new users via the management API."`
	APIListener             string   `envconfig:"API_LISTENER" required:"true" basic:"127.0.0.1:8080" ssl:"127.0.0.1:8080" description:"Network listener for management API binds to. Only 1 listener can be specified. (Default 127.0.0.1 restricts to same machine only)."`
	DBPath                  string   `envconfig:"DB_PATH" required:"true" basic:"go-icq.sqlite" ssl:"go-icq.sqlite" description:"The path to the SQLite database file. The file and DB schema are auto-created if they doesn't exist."`
	LogLevel                string   `envconfig:"LOG_LEVEL" required:"true" basic:"info" ssl:"info" description:"Set logging granularity. Possible values: 'trace', 'debug', 'info', 'warn', 'error'."`
}

func (c *Config) Validate() error {
	// validate TOCListeners
	// (format: hostname:port pairs)
	for _, listener := range c.TOCListeners {
		listener = strings.TrimSpace(listener)
		if listener == "" {
			continue
		}

		if host, port, err := net.SplitHostPort(listener); err != nil {
			return fmt.Errorf("invalid TOC listener %q: %v. Valid format: HOST:PORT (e.g., 0.0.0.0:9898)", listener, err)
		} else if host == "" {
			return fmt.Errorf("invalid TOC listener %q: missing host. Valid format: HOST:PORT (e.g., 0.0.0.0:9898)", listener)
		} else if port == "" {
			return fmt.Errorf("invalid TOC listener %q: missing port. Valid format: HOST:PORT (e.g., 0.0.0.0:9898)", listener)
		}
	}

	// validate APIListener
	// (format: hostname:port pair, no scheme)
	apiListener := strings.TrimSpace(c.APIListener)
	if apiListener == "" {
		return fmt.Errorf("APIListener is required and cannot be empty")
	}

	if host, port, err := net.SplitHostPort(apiListener); err != nil {
		return fmt.Errorf("invalid API listener %q: %v. Valid format: HOST:PORT (e.g., 127.0.0.1:8080)", c.APIListener, err)
	} else if host == "" {
		return fmt.Errorf("invalid API listener %q: missing host. Valid format: HOST:PORT (e.g., 127.0.0.1:8080)", c.APIListener)
	} else if port == "" {
		return fmt.Errorf("invalid API listener %q: missing port. Valid format: HOST:PORT (e.g., 127.0.0.1:8080)", c.APIListener)
	}

	return nil
}

func (c *Config) ParseListenersCfg() ([]Listener, error) {
	m := make(map[string]*Listener)
	// parse BOS listeners
	for _, uriStr := range c.BOSListeners {
		u, err := parseURI(uriStr)
		if err != nil {
			return nil, err
		} else if u == nil {
			continue
		}

		if _, ok := m[u.Scheme]; !ok {
			m[u.Scheme] = &Listener{}
		}

		if m[u.Scheme].BOSListenAddress != "" {
			return nil, errDuplicateListener
		}

		m[u.Scheme].BOSListenAddress = net.JoinHostPort(u.Hostname(), u.Port())
	}

	// parse plaintext BOS advertised listeners
	for _, uriStr := range c.BOSAdvertisedHostsPlain {
		u, err := parseURI(uriStr)
		if err != nil {
			return nil, err
		} else if u == nil {
			continue
		}

		if _, ok := m[u.Scheme]; !ok {
			m[u.Scheme] = &Listener{}
		}

		if m[u.Scheme].BOSAdvertisedHostPlain != "" {
			return nil, errDuplicateListener
		}

		m[u.Scheme].BOSAdvertisedHostPlain = net.JoinHostPort(u.Hostname(), u.Port())
	}

	// parse SSL BOS advertised listeners
	for _, uriStr := range c.BOSAdvertisedHostsSSL {
		u, err := parseURI(uriStr)
		if err != nil {
			return nil, err
		} else if u == nil {
			continue
		}

		if _, ok := m[u.Scheme]; !ok {
			m[u.Scheme] = &Listener{}
		}

		if m[u.Scheme].BOSAdvertisedHostSSL != "" {
			return nil, errDuplicateListener
		}

		m[u.Scheme].HasSSL = true
		m[u.Scheme].BOSAdvertisedHostSSL = net.JoinHostPort(u.Hostname(), u.Port())
	}

	// parse Kerberos listeners
	for _, uriStr := range c.KerberosListeners {
		u, err := parseURI(uriStr)
		if err != nil {
			return nil, err
		}

		if u == nil {
			continue
		}

		if _, ok := m[u.Scheme]; !ok {
			m[u.Scheme] = &Listener{}
		}

		if m[u.Scheme].KerberosListenAddress != "" {
			return nil, errDuplicateListener
		}

		m[u.Scheme].KerberosListenAddress = net.JoinHostPort(u.Hostname(), u.Port())
	}

	ret := make([]Listener, 0, len(m))
	for k, v := range m {
		switch {
		case v.BOSAdvertisedHostPlain == "":
			return nil, fmt.Errorf("missing BOS advertise address for listener `%s://`", k)
		case v.BOSListenAddress == "":
			return nil, fmt.Errorf("missing BOS listen address for listener `%s://`", k)
		}

		ret = append(ret, *v)
	}

	if len(ret) == 0 {
		return nil, errNoBOSListeners
	}

	return ret, nil
}

// uriFormatError is a custom error type for errors related to URIs.
type uriFormatError struct {
	URI string
	Err error
}

func (e uriFormatError) Error() string {
	return fmt.Sprintf("invalid listener URI %q: %v. Valid format: SCHEME://HOST:PORT (e.g., LOCAL://0.0.0.0:5190)", e.URI, e.Err)
}

// parseURI is a helper function to parse and validate a single URI
func parseURI(uriStr string) (u *url.URL, err error) {
	uriStr = strings.TrimSpace(uriStr)
	if uriStr == "" {
		return
	}

	u, err = url.Parse(uriStr)
	if err != nil {
		return nil, uriFormatError{URI: uriStr, Err: err}
	}

	switch {
	case u.Scheme == "":
		return nil, uriFormatError{URI: uriStr, Err: errors.New("missing scheme")}
	case u.Hostname() == "":
		return nil, uriFormatError{URI: uriStr, Err: errors.New("missing host")}
	case u.Port() == "":
		return nil, uriFormatError{URI: uriStr, Err: errors.New("missing port")}
	}

	return u, nil
}
