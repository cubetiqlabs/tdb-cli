package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
	configpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/config"
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

func printJSON(cmd *cobra.Command, value interface{}) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func printCompactJSON(cmd *cobra.Command, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}

func coerceJSONValue(raw string) interface{} {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var value interface{}
	if err := json.Unmarshal([]byte(trimmed), &value); err == nil {
		return value
	}
	return trimmed
}

func makeAuditLogsPretty(items []clientpkg.AuditLog) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, entry := range items {
		row := map[string]any{
			"id":               entry.ID,
			"tenant_id":        entry.TenantID,
			"collection_id":    entry.CollectionID,
			"document_id":      entry.DocumentID,
			"document_version": entry.DocumentVersion,
			"operation":        entry.Operation,
			"actor":            entry.Actor,
			"created_at":       entry.CreatedAt,
			"old_data":         coerceJSONValue(entry.OldData),
			"new_data":         coerceJSONValue(entry.NewData),
		}
		result = append(result, row)
	}
	return result
}

func makeDocumentPretty(doc clientpkg.Document) map[string]any {
	row := map[string]any{
		"id":            doc.ID,
		"tenant_id":     doc.TenantID,
		"collection_id": doc.CollectionID,
		"key":           doc.Key,
		"key_numeric":   doc.KeyNumeric,
		"version":       doc.Version,
		"created_at":    doc.CreatedAt,
		"updated_at":    doc.UpdatedAt,
		"deleted_at":    doc.DeletedAt,
		"data":          coerceJSONValue(doc.Data),
	}
	return row
}

func makeDocumentListPretty(resp *clientpkg.DocumentListResponse) map[string]any {
	items := make([]map[string]any, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, makeDocumentPretty(item))
	}
	return map[string]any{
		"items":      items,
		"pagination": resp.Pagination,
	}
}

func makeDocumentBulkPretty(resp *clientpkg.DocumentBulkResponse) map[string]any {
	items := make([]map[string]any, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, makeDocumentPretty(item))
	}
	return map[string]any{"items": items}
}

func readFileContent(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", nil
	}
	raw, err := os.ReadFile(filepath.Clean(trimmed))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func readJSONPayload(cmd *cobra.Command, inline, filePath string, useStdin, expectArray bool) ([]byte, error) {
	sources := 0
	if strings.TrimSpace(inline) != "" {
		sources++
	}
	if strings.TrimSpace(filePath) != "" {
		sources++
	}
	if useStdin {
		sources++
	}
	if sources == 0 {
		return nil, errors.New("provide --data, --file, or --stdin")
	}
	if sources > 1 {
		return nil, errors.New("use only one of --data, --file, or --stdin")
	}

	var payload []byte
	switch {
	case strings.TrimSpace(inline) != "":
		payload = []byte(inline)
	case strings.TrimSpace(filePath) != "":
		content, err := os.ReadFile(filepath.Clean(filePath))
		if err != nil {
			return nil, err
		}
		payload = content
	case useStdin:
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, err
		}
		payload = data
	}

	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" {
		return nil, errors.New("payload cannot be empty")
	}
	if expectArray && !strings.HasPrefix(trimmed, "[") {
		return nil, errors.New("expected JSON array payload")
	}
	if !json.Valid([]byte(trimmed)) {
		return nil, errors.New("invalid JSON payload")
	}
	return []byte(trimmed), nil
}

func summarizePrimaryKey(field, typ string, auto bool) string {
	if strings.TrimSpace(field) == "" {
		return "-"
	}
	summary := field
	if strings.TrimSpace(typ) != "" {
		summary += fmt.Sprintf(" (%s)", typ)
	}
	if auto {
		summary += " auto"
	}
	return summary
}

func summarizeJSON(raw string, max int) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "-"
	}
	if max <= 0 || len([]rune(trimmed)) <= max {
		return trimmed
	}
	runes := []rune(trimmed)
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
