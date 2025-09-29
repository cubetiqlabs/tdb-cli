package cli

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/config"
)

func TestConfigSetAPIKey(t *testing.T) {
	cases := []struct {
		name          string
		tenantID      string
		tenantName    string
		appID         string
		appName       string
		keyPrefix     string
		scope         string
		expectedAlias string
	}{
		{
			name:          "with key prefix",
			tenantID:      "tenant-alpha",
			tenantName:    "Alpha Corp",
			appID:         "app-main",
			appName:       "Main App",
			keyPrefix:     "alpha",
			scope:         "tenant",
			expectedAlias: "alpha",
		},
		{
			name:          "fallback to app id",
			tenantID:      "tenant-beta",
			tenantName:    "Beta LLC",
			appID:         "app-beta",
			appName:       "Beta App",
			keyPrefix:     "",
			scope:         "application",
			expectedAlias: "app-beta",
		},
		{
			name:          "default alias",
			tenantID:      "tenant-gamma",
			tenantName:    "Gamma Org",
			appID:         "",
			appName:       "",
			keyPrefix:     "",
			scope:         "tenant",
			expectedAlias: "default",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiKey := "test-api-key"
			var receivedKey string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/me" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if r.Method != http.MethodGet {
					t.Fatalf("unexpected method: %s", r.Method)
				}
				receivedKey = r.Header.Get("X-API-Key")
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"tenant_id":"%s","tenant_name":"%s","app_id":"%s","app_name":"%s","status":"active","scope":"%s","key_prefix":"%s"}`,
					tc.tenantID, tc.tenantName, tc.appID, tc.appName, tc.scope, tc.keyPrefix)
			}))
			defer server.Close()

			cfgPath := filepath.Join(t.TempDir(), "config.yaml")
			env := &Environment{
				ConfigPath: cfgPath,
				Config: &configpkg.Config{
					Endpoint: server.URL,
				},
			}

			cmd := newConfigSetCommand(env)
			cmd.SetArgs([]string{"api-key", apiKey})
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)

			if err := cmd.Execute(); err != nil {
				t.Fatalf("cmd.Execute() error = %v, output: %s", err, out.String())
			}

			if receivedKey != apiKey {
				t.Fatalf("expected api key header %q, got %q", apiKey, receivedKey)
			}

			cfg, err := configpkg.Load(cfgPath)
			if err != nil {
				t.Fatalf("load config: %v", err)
			}

			if cfg.DefaultTenant != tc.tenantID {
				t.Fatalf("default tenant = %q, want %q", cfg.DefaultTenant, tc.tenantID)
			}

			tenantCfg, ok := cfg.Tenants[tc.tenantID]
			if !ok {
				t.Fatalf("tenant %q not stored in config", tc.tenantID)
			}

			if tenantCfg.DefaultKey != tc.expectedAlias {
				t.Fatalf("default key alias = %q, want %q", tenantCfg.DefaultKey, tc.expectedAlias)
			}

			entry, ok := tenantCfg.Keys[tc.expectedAlias]
			if !ok {
				t.Fatalf("expected key alias %q stored", tc.expectedAlias)
			}

			if entry.Key != apiKey {
				t.Fatalf("stored api key = %q, want %q", entry.Key, apiKey)
			}

			if entry.AppID != strings.TrimSpace(tc.appID) {
				t.Fatalf("stored app id = %q, want %q", entry.AppID, tc.appID)
			}

			if entry.Prefix != strings.TrimSpace(tc.keyPrefix) {
				t.Fatalf("stored prefix = %q, want %q", entry.Prefix, tc.keyPrefix)
			}

			expectedDescription := strings.TrimSpace(tc.appName)
			if expectedDescription == "" {
				expectedDescription = strings.TrimSpace(tc.scope)
			}

			if entry.Description != expectedDescription {
				t.Fatalf("stored description = %q, want %q", entry.Description, expectedDescription)
			}

			if strings.TrimSpace(tenantCfg.Name) != strings.TrimSpace(tc.tenantName) {
				t.Fatalf("stored tenant name = %q, want %q", tenantCfg.Name, tc.tenantName)
			}

			if !strings.Contains(out.String(), tc.expectedAlias) {
				t.Fatalf("command output %q does not mention alias %q", out.String(), tc.expectedAlias)
			}
		})
	}
}
