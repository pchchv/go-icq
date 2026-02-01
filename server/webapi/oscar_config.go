package webapi

import "github.com/pchchv/go-icq/config"

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
