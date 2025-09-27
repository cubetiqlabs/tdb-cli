package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	clientpkg "cubetiqlabs/tinydb/pkg/tdbcli/client"
)

func newTenantQueriesListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			docs, err := tenantClient.ListSavedQueries(cmd.Context(), auth.appID)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, docs)
			}
			if len(docs) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No saved queries found")
				return nil
			}
			rows := make([][]string, 0, len(docs))
			for _, doc := range docs {
				sq, err := parseSavedQueryDocument(doc)
				name := doc.ID
				qType := "-"
				collection := "-"
				if err == nil {
					name = sq.Name
					if strings.TrimSpace(name) == "" {
						name = doc.ID
					}
					if trimmed := strings.TrimSpace(sq.Type); trimmed != "" {
						qType = trimmed
					}
					if trimmed := strings.TrimSpace(sq.Collection); trimmed != "" {
						collection = trimmed
					}
				}
				rows = append(rows, []string{
					name,
					qType,
					collection,
					formatRelativeTime(doc.UpdatedAt, "-"),
					doc.ID,
				})
			}
			renderTable(cmd, []string{"NAME", "TYPE", "COLLECTION", "UPDATED", "ID"}, rows)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}

func newTenantQueriesGetCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool
	var byName bool
	cmd := &cobra.Command{
		Use:   "get <id_or_name>",
		Short: "Fetch a saved query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			target := strings.TrimSpace(args[0])
			if target == "" {
				return errors.New("identifier cannot be empty")
			}
			var doc *clientpkg.Document
			if byName {
				doc, err = tenantClient.GetSavedQueryByName(cmd.Context(), target, auth.appID)
			} else {
				doc, err = tenantClient.GetSavedQuery(cmd.Context(), target, auth.appID)
			}
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, doc)
			}
			sq, err := parseSavedQueryDocument(*doc)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			collection := strings.TrimSpace(sq.Collection)
			if collection == "" {
				collection = "-"
			}
			fmt.Fprintf(out, "ID: %s\n", doc.ID)
			fmt.Fprintf(out, "NAME: %s\n", sq.Name)
			fmt.Fprintf(out, "TYPE: %s\n", sq.Type)
			fmt.Fprintf(out, "COLLECTION: %s\n", collection)
			fmt.Fprintf(out, "CREATED: %s\n", formatTime(doc.CreatedAt))
			fmt.Fprintf(out, "UPDATED: %s\n", formatTime(doc.UpdatedAt))
			if doc.DeletedAt != nil {
				fmt.Fprintf(out, "DELETED: %s\n", formatTime(*doc.DeletedAt))
			}
			switch strings.ToLower(strings.TrimSpace(sq.Type)) {
			case "dsl":
				fmt.Fprintln(out, "DSL:")
				if len(sq.DSL) == 0 {
					fmt.Fprintln(out, "  (empty)")
					return nil
				}
				return printJSON(cmd, jsonStringToInterface(string(sq.DSL)))
			case "sql":
				fmt.Fprintln(out, "SQL:")
				sql := strings.TrimSpace(sq.SQL)
				if sql == "" {
					fmt.Fprintln(out, "  (empty)")
					return nil
				}
				fmt.Fprintln(out, sql)
				return nil
			default:
				fmt.Fprintln(out, "PAYLOAD:")
				return printJSON(cmd, jsonStringToInterface(doc.Data))
			}
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON document")
	cmd.Flags().BoolVar(&byName, "by-name", false, "Treat the identifier as the saved query name")
	return cmd
}

func newTenantQueriesCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create or upsert a saved query",
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			doc, err := tenantClient.CreateSavedQuery(cmd.Context(), payload, auth.appID)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, doc)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved query stored with ID %s\n", doc.ID)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON saved query payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to saved query JSON payload")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read saved query JSON payload from stdin")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}

func newTenantQueriesPutCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool
	cmd := &cobra.Command{
		Use:   "put <name>",
		Short: "Replace a saved query by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("name cannot be empty")
			}
			doc, err := tenantClient.PutSavedQuery(cmd.Context(), name, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, doc)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved query %s replaced\n", name)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON saved query payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to saved query JSON payload")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read saved query JSON payload from stdin")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}

func newTenantQueriesPatchCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var data string
	var file string
	var stdin bool
	var raw bool
	cmd := &cobra.Command{
		Use:   "patch <name>",
		Short: "Patch fields on a saved query by name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			payload, err := readJSONPayload(cmd, data, file, stdin, false)
			if err != nil {
				return err
			}
			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("name cannot be empty")
			}
			doc, err := tenantClient.PatchSavedQuery(cmd.Context(), name, payload, auth.appID)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, doc)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved query %s patched\n", name)
			return nil
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON patch payload")
	cmd.Flags().StringVar(&file, "file", "", "Path to saved query patch payload")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Read patch payload from stdin")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}

func newTenantQueriesExecuteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var params string
	var paramsFile string
	var paramsStdin bool
	var byName bool
	var raw bool
	cmd := &cobra.Command{
		Use:   "execute <id_or_name>",
		Short: "Execute a saved query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			target := strings.TrimSpace(args[0])
			if target == "" {
				return errors.New("identifier cannot be empty")
			}
			var payload []byte
			if cmd.Flags().Lookup("params").Changed || cmd.Flags().Lookup("params-file").Changed || cmd.Flags().Lookup("params-stdin").Changed {
				payload, err = readJSONPayload(cmd, params, paramsFile, paramsStdin, false)
				if err != nil {
					return err
				}
			}
			var result *clientpkg.SavedQueryExecutionResult
			if byName {
				result, err = tenantClient.ExecuteSavedQueryByName(cmd.Context(), target, payload, auth.appID)
			} else {
				result, err = tenantClient.ExecuteSavedQueryByID(cmd.Context(), target, payload, auth.appID)
			}
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, result)
			}
			return renderSavedQueryResult(cmd, result)
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().StringVar(&params, "params", "", "Inline JSON parameters for execution (wrapped in {\"params\":{...}})")
	cmd.Flags().StringVar(&paramsFile, "params-file", "", "Path to JSON parameters for execution")
	cmd.Flags().BoolVar(&paramsStdin, "params-stdin", false, "Read JSON parameters from stdin")
	cmd.Flags().BoolVar(&byName, "by-name", false, "Execute using the saved query name")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON result")
	return cmd
}

func newTenantQueriesDeleteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var byName bool
	var purge bool
	var confirm bool

	cmd := &cobra.Command{
		Use:   "delete <id_or_name>",
		Short: "Delete or purge a saved query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			target := strings.TrimSpace(args[0])
			if target == "" {
				return errors.New("identifier cannot be empty")
			}
			if purge && !confirm {
				return errors.New("use --confirm to acknowledge irreversible purge")
			}
			if byName {
				if err := tenantClient.DeleteSavedQueryByName(cmd.Context(), target, purge, auth.appID, confirm); err != nil {
					return err
				}
			} else {
				if err := tenantClient.DeleteSavedQueryByID(cmd.Context(), target, purge, auth.appID, confirm); err != nil {
					return err
				}
			}
			verb := "Deleted"
			if purge {
				verb = "Purged"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s saved query %s\n", verb, target)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&byName, "by-name", false, "Treat the identifier as the saved query name")
	cmd.Flags().BoolVar(&purge, "purge", false, "Permanently purge the saved query document")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Confirm irreversible purge")
	return cmd
}

func newTenantQueriesParamsTemplateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var byName bool
	var outPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "params-template <id_or_name>",
		Short: "Generate an execution params template for a saved query",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			target := strings.TrimSpace(args[0])
			if target == "" {
				return errors.New("identifier cannot be empty")
			}
			var doc *clientpkg.Document
			if byName {
				doc, err = tenantClient.GetSavedQueryByName(cmd.Context(), target, auth.appID)
			} else {
				doc, err = tenantClient.GetSavedQuery(cmd.Context(), target, auth.appID)
			}
			if err != nil {
				return err
			}
			sq, err := parseSavedQueryDocument(*doc)
			if err != nil {
				return err
			}
			template := buildParamsTemplate(sq)
			payload := map[string]any{"params": template}
			pretty, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				return err
			}
			targetPath := strings.TrimSpace(outPath)
			if targetPath == "" {
				fmt.Fprintln(cmd.OutOrStdout(), string(pretty))
				return nil
			}
			clean := filepath.Clean(targetPath)
			if !force {
				if _, err := os.Stat(clean); err == nil {
					return fmt.Errorf("file %s already exists (use --force to overwrite)", clean)
				}
			}
			dir := filepath.Dir(clean)
			if dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return err
				}
			}
			if err := os.WriteFile(clean, append(pretty, '\n'), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote params template to %s\n", clean)
			return nil
		},
	}

	auth.bindWithApp(cmd)
	cmd.Flags().BoolVar(&byName, "by-name", false, "Generate template using the saved query name")
	cmd.Flags().StringVar(&outPath, "out", "", "Optional path to write the params template")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite the output file if it exists")
	return cmd
}

func parseSavedQueryDocument(doc clientpkg.Document) (clientpkg.SavedQuery, error) {
	var sq clientpkg.SavedQuery
	trimmed := strings.TrimSpace(doc.Data)
	if trimmed == "" {
		return sq, errors.New("saved query payload empty")
	}
	if err := json.Unmarshal([]byte(trimmed), &sq); err != nil {
		return sq, err
	}
	if sq.Name == "" {
		sq.Name = doc.ID
	}
	return sq, nil
}

func renderSavedQueryResult(cmd *cobra.Command, result *clientpkg.SavedQueryExecutionResult) error {
	if result == nil || len(result.Items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No rows returned")
		return nil
	}
	columns := make(map[string]struct{})
	for _, row := range result.Items {
		for key := range row {
			columns[key] = struct{}{}
		}
	}
	headers := make([]string, 0, len(columns))
	for key := range columns {
		headers = append(headers, key)
	}
	sort.Strings(headers)
	rows := make([][]string, 0, len(result.Items))
	for _, row := range result.Items {
		cells := make([]string, len(headers))
		for i, header := range headers {
			cells[i] = stringifyValue(row[header])
		}
		rows = append(rows, cells)
	}
	renderTable(cmd, headers, rows)
	return nil
}

func stringifyValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	case []any, map[string]any:
		raw, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprint(val)
		}
		return summarizeJSON(string(raw), 80)
	default:
		return fmt.Sprint(val)
	}
}

func buildParamsTemplate(sq clientpkg.SavedQuery) map[string]any {
	template := make(map[string]any)
	if strings.EqualFold(sq.Type, "sql") {
		for _, placeholder := range extractSQLParams(sq.SQL) {
			template[placeholder] = ""
		}
		return template
	}
	if strings.EqualFold(sq.Type, "dsl") && len(sq.DSL) > 0 {
		template = extractDSLParams(sq.DSL)
	}
	if len(template) == 0 {
		return map[string]any{}
	}
	return template
}

func extractSQLParams(sql string) []string {
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(sql, -1)
	seen := make(map[string]struct{})
	ordered := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		name := match[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		ordered = append(ordered, name)
	}
	return ordered
}

func extractDSLParams(raw json.RawMessage) map[string]any {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{}
	}
	params := make(map[string]any)
	if limit, ok := payload["limit"].(float64); ok {
		params["limit"] = int(limit)
	}
	if offset, ok := payload["offset"].(float64); ok {
		params["offset"] = int(offset)
	}
	if cursor := strings.TrimSpace(stringifyValue(payload["cursor"])); cursor != "" {
		params["cursor"] = cursor
	}
	if orderBy, ok := payload["orderBy"].([]any); ok {
		params["orderBy"] = orderBy
	}
	if selectFields, ok := payload["select"].([]any); ok {
		params["select"] = selectFields
	}
	filters := collectFilters(payload)
	if len(filters) > 0 {
		params["filters"] = filters
	}
	return params
}

func collectFilters(payload map[string]any) map[string]any {
	filters := make(map[string]any)
	where, ok := payload["where"].(map[string]any)
	if !ok {
		return filters
	}
	extractConditions := func(arr []any) {
		for _, item := range arr {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			for field, value := range m {
				if inner, ok := value.(map[string]any); ok {
					if eqVal, ok := inner["eq"]; ok {
						filters[field] = eqVal
						continue
					}
					if inVal, ok := inner["in"]; ok {
						filters[field] = inVal
						continue
					}
					if gte, okGte := inner["gte"]; okGte {
						filters[field+"_from"] = gte
					}
					if lte, okLte := inner["lte"]; okLte {
						filters[field+"_to"] = lte
					}
					continue
				}
				filters[field] = value
			}
		}
	}
	if andArr, ok := where["and"].([]any); ok {
		extractConditions(andArr)
	}
	if orArr, ok := where["or"].([]any); ok {
		extractConditions(orArr)
	}
	return filters
}
