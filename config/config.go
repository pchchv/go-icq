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

type Config struct {
	BOSListeners            []string
	BOSAdvertisedHostsPlain []string
	BOSAdvertisedHostsSSL   []string
	KerberosListeners       []string
	TOCListeners            []string
	DisableAuth             bool
	APIListener             string
	DBPath                  string
	LogLevel                string
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
