package config

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
