package config

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
