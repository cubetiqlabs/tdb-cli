package config

import "strings"

// MaskedClone returns a deep copy of the configuration with all secrets masked.
func (c *Config) MaskedClone() *Config {
	if c == nil {
		return nil
	}

	clone := &Config{
		Endpoint:      c.Endpoint,
		AdminSecret:   c.MaskedAdminSecret(),
		DefaultTenant: c.DefaultTenant,
	}

	if c.Tenants != nil {
		clone.Tenants = make(map[string]TenantConfig, len(c.Tenants))
		for id, tc := range c.Tenants {
			tcClone := TenantConfig{
				Name:       tc.Name,
				DefaultKey: tc.DefaultKey,
			}
			if tc.Keys != nil {
				tcClone.Keys = make(map[string]APIKeyEntry, len(tc.Keys))
				for keyAlias, entry := range tc.Keys {
					maskedEntry := entry
					maskedEntry.Key = maskAPIKey(entry.Key)
					tcClone.Keys[keyAlias] = maskedEntry
				}
			}
			clone.Tenants[id] = tcClone
		}
	}

	return clone
}

// maskAPIKey masks an API key string, showing only the first 3 and last 3 characters.
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 6 {
		return strings.Repeat("*", len(key))
	}
	return key[:3] + strings.Repeat("*", len(key)-6) + key[len(key)-3:]
}
