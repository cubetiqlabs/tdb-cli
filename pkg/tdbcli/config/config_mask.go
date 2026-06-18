package config

import "strings"

// MaskedClone returns a deep clone of the Config with all secrets properly masked.
// This prevents credential leakage when serializing the configuration.
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
		for tID, tc := range c.Tenants {
			tcClone := TenantConfig{
				Name:       tc.Name,
				DefaultKey: tc.DefaultKey,
			}
			if tc.Keys != nil {
				tcClone.Keys = make(map[string]APIKeyEntry, len(tc.Keys))
				for kID, keyEntry := range tc.Keys {
					keyEntryClone := keyEntry
					if keyEntryClone.Key != "" {
						if len(keyEntryClone.Key) <= 8 {
							keyEntryClone.Key = strings.Repeat("*", len(keyEntryClone.Key))
						} else {
							keyEntryClone.Key = keyEntryClone.Key[:4] + strings.Repeat("*", len(keyEntryClone.Key)-8) + keyEntryClone.Key[len(keyEntryClone.Key)-4:]
						}
					}
					tcClone.Keys[kID] = keyEntryClone
				}
			}
			clone.Tenants[tID] = tcClone
		}
	}
	return clone
}
