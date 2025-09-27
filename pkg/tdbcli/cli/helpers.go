package cli

import (
	"errors"
	"fmt"
	"strings"

	clientpkg "cubetiqlabs/tinydb/pkg/tdbcli/client"
	configpkg "cubetiqlabs/tinydb/pkg/tdbcli/config"
)

func requireEnvironment(cmdEnv *Environment) (*Environment, error) {
	if cmdEnv == nil {
		return nil, errors.New("cli environment is nil")
	}
	if cmdEnv.Config == nil {
		return nil, errors.New("configuration not loaded; ensure command runs after initialization")
	}
	return cmdEnv, nil
}

func ensureEndpoint(env *Environment) (string, error) {
	env, err := requireEnvironment(env)
	if err != nil {
		return "", err
	}
	endpoint := strings.TrimSpace(env.Config.Endpoint)
	if endpoint == "" {
		return "", errors.New("endpoint not configured; run `tdb config set endpoint <url>`")
	}
	return endpoint, nil
}

func resolveTenantID(env *Environment, tenantID string) (string, error) {
	envCtx, err := requireEnvironment(env)
	if err != nil {
		return "", err
	}
	resolved := strings.TrimSpace(tenantID)
	if resolved == "" {
		resolved = strings.TrimSpace(envCtx.Config.DefaultTenant)
	}
	if resolved == "" {
		return "", errors.New("--tenant is required (set a default via `tdb config set default-tenant <tenant_id>`)")
	}
	return resolved, nil
}

func adminClientFromEnv(env *Environment) (*clientpkg.AdminClient, error) {
	endpoint, err := ensureEndpoint(env)
	if err != nil {
		return nil, err
	}
	secret := strings.TrimSpace(env.Config.AdminSecret)
	if secret == "" {
		return nil, errors.New("admin secret not configured; run `tdb config set admin-secret <secret>`")
	}
	return clientpkg.NewAdminClient(endpoint, secret)
}

func tenantClientFromEnv(env *Environment, tenantID, keyName, apiKeyOverride string) (*clientpkg.TenantClient, configpkg.APIKeyEntry, error) {
	endpoint, err := ensureEndpoint(env)
	if err != nil {
		return nil, configpkg.APIKeyEntry{}, err
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, configpkg.APIKeyEntry{}, errors.New("tenant id is required")
	}

	var entry configpkg.APIKeyEntry
	if key := strings.TrimSpace(apiKeyOverride); key != "" {
		entry = configpkg.APIKeyEntry{Key: key}
	} else {
		cfg, err := requireEnvironment(env)
		if err != nil {
			return nil, configpkg.APIKeyEntry{}, err
		}
		resolved, err := cfg.Config.ResolveKey(tenantID, keyName)
		if err != nil {
			return nil, configpkg.APIKeyEntry{}, err
		}
		entry = resolved
	}
	if strings.TrimSpace(entry.Key) == "" {
		return nil, configpkg.APIKeyEntry{}, errors.New("api key is empty")
	}
	tenantClient, err := clientpkg.NewTenantClient(endpoint, entry.Key)
	if err != nil {
		return nil, configpkg.APIKeyEntry{}, err
	}
	return tenantClient, entry, nil
}

func storeAPIKey(env *Environment, tenantID, alias string, entry configpkg.APIKeyEntry, setDefault bool, tenantName string) error {
	cfgEnv, err := requireEnvironment(env)
	if err != nil {
		return err
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return errors.New("key alias cannot be empty")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return errors.New("tenant id cannot be empty")
	}
	if strings.TrimSpace(entry.Key) == "" {
		return errors.New("api key value cannot be empty")
	}

	cfg := cfgEnv.Config
	tc := cfg.EnsureTenant(tenantID)
	if tenantName != "" {
		tc.Name = tenantName
	} else if tc.Name == "" {
		tc.Name = tenantID
	}
	if tc.Keys == nil {
		tc.Keys = make(map[string]configpkg.APIKeyEntry)
	}
	tc.Keys[alias] = entry
	if setDefault {
		tc.DefaultKey = alias
	}
	cfg.UpdateTenant(tenantID, tc)
	if setDefault || strings.TrimSpace(cfg.DefaultTenant) == "" {
		cfg.DefaultTenant = tenantID
	}
	return env.Save()
}

func deleteAPIKey(env *Environment, tenantID, alias string) (bool, error) {
	cfgEnv, err := requireEnvironment(env)
	if err != nil {
		return false, err
	}
	tenantID = strings.TrimSpace(tenantID)
	alias = strings.TrimSpace(alias)
	if tenantID == "" || alias == "" {
		return false, errors.New("tenant id and key alias are required")
	}
	cfg := cfgEnv.Config
	tc, ok := cfg.Tenants[tenantID]
	if !ok || tc.Keys == nil {
		return false, fmt.Errorf("tenant %s has no stored keys", tenantID)
	}
	if _, ok := tc.Keys[alias]; !ok {
		return false, fmt.Errorf("key %s not found for tenant %s", alias, tenantID)
	}
	delete(tc.Keys, alias)
	if tc.DefaultKey == alias {
		tc.DefaultKey = ""
	}
	if len(tc.Keys) == 0 && tc.Name == "" && tc.DefaultKey == "" {
		delete(cfg.Tenants, tenantID)
	} else {
		cfg.UpdateTenant(tenantID, tc)
	}
	if cfg.DefaultTenant == tenantID {
		if _, ok := cfg.Tenants[tenantID]; !ok {
			cfg.DefaultTenant = ""
		}
	}
	if err := env.Save(); err != nil {
		return false, err
	}
	return true, nil
}

func setDefaultKey(env *Environment, tenantID, alias string) error {
	cfgEnv, err := requireEnvironment(env)
	if err != nil {
		return err
	}
	tenantID = strings.TrimSpace(tenantID)
	alias = strings.TrimSpace(alias)
	if tenantID == "" || alias == "" {
		return errors.New("tenant id and key alias are required")
	}
	cfg := cfgEnv.Config
	tc, ok := cfg.Tenants[tenantID]
	if !ok || tc.Keys == nil {
		return fmt.Errorf("tenant %s has no stored keys", tenantID)
	}
	if _, ok := tc.Keys[alias]; !ok {
		return fmt.Errorf("key %s not found for tenant %s", alias, tenantID)
	}
	tc.DefaultKey = alias
	cfg.UpdateTenant(tenantID, tc)
	if strings.TrimSpace(cfg.DefaultTenant) == "" {
		cfg.DefaultTenant = tenantID
	}
	return env.Save()
}

func setDefaultTenant(env *Environment, tenantID string) error {
	cfgEnv, err := requireEnvironment(env)
	if err != nil {
		return err
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return errors.New("tenant id is required")
	}
	cfg := cfgEnv.Config
	cfg.EnsureTenant(tenantID)
	cfg.DefaultTenant = tenantID
	return env.Save()
}
