package config

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config with all fields",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898", "192.168.1.10:9899"},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr: false,
		},
		{
			name: "valid config with single TOC listener",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898"},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr: false,
		},
		{
			name: "valid config with empty TOC listeners",
			config: Config{
				TOCListeners: []string{},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr: false,
		},
		{
			name: "valid config with empty API listener",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898"},
				APIListener:  "",
			},
			wantErr:     true,
			errContains: "APIListener is required and cannot be empty",
		},
		{
			name: "valid config with all empty",
			config: Config{
				TOCListeners: []string{},
				APIListener:  "",
			},
			wantErr:     true,
			errContains: "APIListener is required and cannot be empty",
		},
		{
			name: "invalid TOC listener - missing port",
			config: Config{
				TOCListeners: []string{"0.0.0.0"},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr:     true,
			errContains: "invalid TOC listener \"0.0.0.0\": address 0.0.0.0: missing port in address",
		},
		{
			name: "invalid TOC listener - missing host",
			config: Config{
				TOCListeners: []string{":9898"},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr:     true,
			errContains: "invalid TOC listener \":9898\": missing host",
		},
		{
			name: "invalid TOC listener - malformed",
			config: Config{
				TOCListeners: []string{"invalid-format"},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr:     true,
			errContains: "invalid TOC listener \"invalid-format\": address invalid-format: missing port in address",
		},
		{
			name: "invalid TOC listener in comma-separated list",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898", "invalid-format", "192.168.1.10:9899"},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr:     true,
			errContains: "invalid TOC listener \"invalid-format\": address invalid-format: missing port in address",
		},
		{
			name: "invalid API listener - missing port",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898"},
				APIListener:  "127.0.0.1",
			},
			wantErr:     true,
			errContains: "invalid API listener \"127.0.0.1\": address 127.0.0.1: missing port in address",
		},
		{
			name: "invalid API listener - missing host",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898"},
				APIListener:  ":8080",
			},
			wantErr:     true,
			errContains: "invalid API listener \":8080\": missing host",
		},
		{
			name: "invalid API listener - malformed",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898"},
				APIListener:  "invalid-format",
			},
			wantErr:     true,
			errContains: "invalid API listener \"invalid-format\": address invalid-format: missing port in address",
		},
		{
			name: "whitespace-only TOC listeners",
			config: Config{
				TOCListeners: []string{"   ", "  ", "  "},
				APIListener:  "127.0.0.1:8080",
			},
			wantErr: false,
		},
		{
			name: "whitespace-only API listener",
			config: Config{
				TOCListeners: []string{"0.0.0.0:9898"},
				APIListener:  "   ",
			},
			wantErr:     true,
			errContains: "APIListener is required and cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error but got none")
					return
				}

				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Config.Validate() error = %v, want error containing %q", err, tt.errContains)
				}

				return
			}

			if err != nil {
				t.Errorf("Config.Validate() unexpected error = %v", err)
			}
		})
	}
}

// contains it's a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}

	if len(s) < len(substr) {
		return false
	}

	if s == substr {
		return true
	}

	if s[:len(substr)] == substr {
		return true
	}

	if s[len(s)-len(substr):] == substr {
		return true
	}

	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
