package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
)

var supportedAuditOperations = map[string]struct{}{
	"create": {},
	"update": {},
	"patch":  {},
	"delete": {},
	"purge":  {},
}

func newTenantAuditCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var limit int
	var collectionFilter string
	var documentFilter string
	var operationFilter string
	var sinceStr string
	var untilStr string
	var actorFilter string
	var raw bool
	var rawPretty bool
	var sortFields []string

	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Inspect audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}

			params := clientpkg.ListAuditLogsParams{
				AppID:      auth.appID,
				Limit:      limit,
				DocumentID: strings.TrimSpace(documentFilter),
			}

			normalizedSort, err := normalizeSortTokens(sortFields)
			if err != nil {
				return err
			}
			params.Sort = normalizedSort

			if trimmed := strings.TrimSpace(operationFilter); trimmed != "" {
				op := strings.ToLower(trimmed)
				if _, ok := supportedAuditOperations[op]; !ok {
					valid := make([]string, 0, len(supportedAuditOperations))
					for k := range supportedAuditOperations {
						valid = append(valid, k)
					}
					sort.Strings(valid)
					return fmt.Errorf("unsupported operation %q (expected one of %s)", trimmed, strings.Join(valid, ", "))
				}
				params.Operation = op
			}

			now := time.Now().UTC()
			if trimmed := strings.TrimSpace(sinceStr); trimmed != "" {
				ts, err := parseAuditTimeArg(trimmed, now)
				if err != nil {
					return fmt.Errorf("invalid --since value %q: %w", trimmed, err)
				}
				params.Since = &ts
			}
			if trimmed := strings.TrimSpace(untilStr); trimmed != "" {
				tu, err := parseAuditTimeArg(trimmed, now)
				if err != nil {
					return fmt.Errorf("invalid --until value %q: %w", trimmed, err)
				}
				params.Until = &tu
			}
			if trimmed := strings.TrimSpace(actorFilter); trimmed != "" {
				params.Actor = trimmed
			}

			collectionNameMap := map[string]string{}
			collections, collectionsErr := tenantClient.ListCollections(cmd.Context(), auth.appID)
			if collectionsErr == nil {
				for _, col := range collections {
					collectionNameMap[col.ID] = col.Name
				}
			}

			if trimmed := strings.TrimSpace(collectionFilter); trimmed != "" {
				if collectionsErr != nil {
					return fmt.Errorf("failed to resolve collection %q: %w", trimmed, collectionsErr)
				}
				resolvedID := ""
				for _, col := range collections {
					if strings.EqualFold(col.ID, trimmed) || strings.EqualFold(col.Name, trimmed) {
						resolvedID = col.ID
						break
					}
				}
				if resolvedID == "" {
					return fmt.Errorf("collection %q not found", trimmed)
				}
				params.CollectionID = resolvedID
			}

			logs, err := tenantClient.ListAuditLogs(cmd.Context(), params)
			if err != nil {
				return err
			}

			if raw || rawPretty {
				if rawPretty {
					payload := map[string]any{"items": makeAuditLogsPretty(logs)}
					return printJSON(cmd, payload)
				}
				payload := clientpkg.AuditLogListResponse{Items: logs}
				return printCompactJSON(cmd, payload)
			}

			if len(logs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No audit entries found")
				return nil
			}

			rows := make([][]string, 0, len(logs))
			for _, entry := range logs {
				collectionLabel := entry.CollectionID
				if name := collectionNameMap[entry.CollectionID]; strings.TrimSpace(name) != "" {
					collectionLabel = name
				}
				docLabel := entry.DocumentID
				if strings.TrimSpace(docLabel) == "" {
					docLabel = "-"
				}
				actor := strings.TrimSpace(entry.Actor)
				if actor == "" {
					actor = "-"
				}
				rows = append(rows, []string{
					formatRelativeTime(entry.CreatedAt, "-"),
					collectionLabel,
					docLabel,
					strings.ToUpper(entry.Operation),
					actor,
					summarizeAuditChange(entry.OldData, entry.NewData),
				})
			}
			renderTable(cmd, []string{"WHEN", "COLLECTION", "DOCUMENT", "OPERATION", "ACTOR", "CHANGE"}, rows)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of audit entries to return (max 500)")
	cmd.Flags().StringVar(&collectionFilter, "collection", "", "Filter by collection name or ID")
	cmd.Flags().StringVar(&documentFilter, "document", "", "Filter by document ID")
	cmd.Flags().StringVar(&operationFilter, "operation", "", "Filter by operation (create, update, patch, delete, purge)")
	cmd.Flags().StringVar(&sinceStr, "since", "", "Only include entries on or after this RFC3339 timestamp")
	cmd.Flags().StringVar(&untilStr, "until", "", "Only include entries on or before this RFC3339 timestamp")
	cmd.Flags().StringVar(&actorFilter, "actor", "", "Filter by actor identifier")
	cmd.Flags().StringSliceVar(&sortFields, "sort", []string{"-created_at"}, "Sort order (comma separated). Prefix with - for descending. Fields: created_at, operation, actor, collection, document_id, id")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print compact JSON response")
	cmd.Flags().BoolVar(&rawPretty, "raw-pretty", false, "Print pretty JSON response")

	return cmd
}

func normalizeSortTokens(tokens []string) ([]string, error) {
	if len(tokens) == 0 {
		tokens = []string{"-created_at"}
	}
	expanded := make([]string, 0, len(tokens))
	for _, v := range tokens {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				expanded = append(expanded, trimmed)
			}
		}
	}
	if len(expanded) == 0 {
		expanded = append(expanded, "-created_at")
	}
	result := make([]string, 0, len(expanded))
	for _, token := range expanded {
		desc := false
		field := token
		if strings.HasPrefix(field, "-") {
			desc = true
			field = field[1:]
		} else if strings.HasPrefix(field, "+") {
			field = field[1:]
		}
		field = strings.ToLower(strings.TrimSpace(field))
		switch field {
		case "created_at", "operation", "actor", "collection", "document_id", "id":
			if desc {
				result = append(result, "-"+field)
			} else {
				result = append(result, field)
			}
		default:
			return nil, fmt.Errorf("unsupported sort field %q", token)
		}
	}
	if len(result) == 0 {
		result = append(result, "-created_at")
	}
	return result, nil
}

func summarizeAuditChange(oldJSON, newJSON string) string {
	oldSummary := summarizeJSON(oldJSON, 28)
	newSummary := summarizeJSON(newJSON, 28)
	switch {
	case oldSummary == "-" && newSummary == "-":
		return "-"
	case oldSummary == "-":
		return "→ " + newSummary
	case newSummary == "-":
		return oldSummary + " → -"
	case oldSummary == newSummary:
		return newSummary
	default:
		return oldSummary + " → " + newSummary
	}
}

func parseAuditTimeArg(raw string, now time.Time) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("value cannot be empty")
	}
	if ts, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return ts, nil
	}
	dur, err := parseFlexibleDurationArg(trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return now.Add(-dur), nil
}

func parseFlexibleDurationArg(raw string) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("duration cannot be empty")
	}
	if d, err := time.ParseDuration(trimmed); err == nil {
		return d, nil
	}
	if strings.HasSuffix(trimmed, "d") {
		value := strings.TrimSuffix(trimmed, "d")
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, err
		}
		hours := f * 24
		return time.Duration(hours * float64(time.Hour)), nil
	}
	return 0, fmt.Errorf("invalid duration %q", raw)
}
