package config

import "strings"

// MaskedClone returns a deep copy of the configuration with all secrets (AdminSecret and API Keys) masked.
// This ensures that displaying the configuration does not leak credentials, and because it is a deep copy,
// modifying the clone does not affect the original maps.
func (c *Config) MaskedClone() Config {
	clone := *c
	clone.AdminSecret = c.MaskedAdminSecret()

	if c.Tenants != nil {
		clone.Tenants = make(map[string]TenantConfig, len(c.Tenants))
		for id, tc := range c.Tenants {
			tcClone := tc
			if tc.Keys != nil {
				tcClone.Keys = make(map[string]APIKeyEntry, len(tc.Keys))
				for keyAlias, entry := range tc.Keys {
					entryClone := entry
					if entryClone.Key != "" {
						if len(entryClone.Key) <= 6 {
							entryClone.Key = strings.Repeat("*", len(entryClone.Key))
						} else {
							entryClone.Key = entryClone.Key[:3] + strings.Repeat("*", len(entryClone.Key)-6) + entryClone.Key[len(entryClone.Key)-3:]
						}
					}
					tcClone.Keys[keyAlias] = entryClone
				}
			}
			clone.Tenants[id] = tcClone
		}
	}

	return clone
}
