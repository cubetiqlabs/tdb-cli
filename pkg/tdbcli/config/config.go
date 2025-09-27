package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents persisted CLI configuration for TinyDB.
type Config struct {
	Endpoint      string                  `yaml:"endpoint"`
	AdminSecret   string                  `yaml:"admin_secret"`
	DefaultTenant string                  `yaml:"default_tenant,omitempty"`
	Tenants       map[string]TenantConfig `yaml:"tenants,omitempty"`
}

// TenantConfig stores API credentials cached for a tenant.
type TenantConfig struct {
	Name       string                 `yaml:"name,omitempty"`
	DefaultKey string                 `yaml:"default_key,omitempty"`
	Keys       map[string]APIKeyEntry `yaml:"keys,omitempty"`
}

// APIKeyEntry stores a named API key for either tenant- or app-scoped access.
type APIKeyEntry struct {
	Key         string `yaml:"key"`
	Prefix      string `yaml:"prefix,omitempty"`
	AppID       string `yaml:"app_id,omitempty"`
	Description string `yaml:"description,omitempty"`
}

// DefaultPath returns the default config file path, creating the parent directory if necessary.
func DefaultPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.Getenv("XDG_CONFIG_HOME")
		if strings.TrimSpace(base) == "" {
			home, homeErr := os.UserHomeDir()
			if homeErr != nil {
				if err != nil {
					return "", err
				}
				return "", homeErr
			}
			base = filepath.Join(home, ".config")
		}
	}
	dir := filepath.Join(base, "tdb")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the configuration from the provided path. If the file is missing, an empty config is returned.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Tenants == nil {
		cfg.Tenants = make(map[string]TenantConfig)
	}
	return cfg, nil
}

// Save writes the configuration to disk, creating parent directories when required.
func (c *Config) Save(path string) error {
	if c.Tenants == nil {
		c.Tenants = make(map[string]TenantConfig)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// EnsureTenant returns the tenant entry, creating it when absent.
func (c *Config) EnsureTenant(id string) TenantConfig {
	if c.Tenants == nil {
		c.Tenants = make(map[string]TenantConfig)
	}
	tc, ok := c.Tenants[id]
	if !ok {
		tc = TenantConfig{Keys: make(map[string]APIKeyEntry)}
	} else if tc.Keys == nil {
		tc.Keys = make(map[string]APIKeyEntry)
	}
	c.Tenants[id] = tc
	return tc
}

// UpdateTenant persists the provided tenant configuration in-memory.
func (c *Config) UpdateTenant(id string, tc TenantConfig) {
	if c.Tenants == nil {
		c.Tenants = make(map[string]TenantConfig)
	}
	if tc.Keys == nil {
		tc.Keys = make(map[string]APIKeyEntry)
	}
	c.Tenants[id] = tc
}

// ResolveKey retrieves an API key for the given tenant. keyName may be empty to use the configured default.
func (c *Config) ResolveKey(tenantID, keyName string) (APIKeyEntry, error) {
	tc, ok := c.Tenants[tenantID]
	if !ok {
		return APIKeyEntry{}, fmt.Errorf("tenant %s not found in config", tenantID)
	}
	if tc.Keys == nil {
		return APIKeyEntry{}, fmt.Errorf("tenant %s has no stored keys", tenantID)
	}
	candidate := keyName
	if candidate == "" {
		candidate = tc.DefaultKey
	}
	if candidate == "" {
		return APIKeyEntry{}, fmt.Errorf("no key name provided and no default configured for tenant %s", tenantID)
	}
	entry, ok := tc.Keys[candidate]
	if !ok {
		return APIKeyEntry{}, fmt.Errorf("key %s not found for tenant %s", candidate, tenantID)
	}
	return entry, nil
}

// MaskedAdminSecret returns a masked representation for display.
func (c *Config) MaskedAdminSecret() string {
	if c.AdminSecret == "" {
		return ""
	}
	if len(c.AdminSecret) <= 6 {
		return strings.Repeat("*", len(c.AdminSecret))
	}
	return c.AdminSecret[:3] + strings.Repeat("*", len(c.AdminSecret)-6) + c.AdminSecret[len(c.AdminSecret)-3:]
}
