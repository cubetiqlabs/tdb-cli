package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// webhookRow and scheduleRow are the fields the tables show; the full answer is
// available with --raw.
type webhookRow struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	URL                  string   `json:"url"`
	Events               []string `json:"events"`
	Enabled              bool     `json:"enabled"`
	TotalDeliveries      int64    `json:"total_deliveries"`
	SuccessfulDeliveries int64    `json:"successful_deliveries"`
	FailedDeliveries     int64    `json:"failed_deliveries"`
	Secret               string   `json:"secret,omitempty"`
}

type deliveryRow struct {
	ID         string `json:"id"`
	WebhookID  string `json:"webhook_id"`
	EventType  string `json:"event_type"`
	Status     string `json:"status"`
	Attempts   int    `json:"attempts"`
	StatusCode int    `json:"status_code"`
	CreatedAt  string `json:"created_at"`
}

func newTenantWebhooksCommand(env *Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhooks",
		Aliases: []string{"webhook"},
		Short:   "Manage webhook subscriptions",
		Long:    "Create and manage webhook subscriptions, rotate their signing secret, and inspect or redeliver deliveries. Needs a tenant-scoped API key.",
	}
	cmd.AddCommand(
		newWebhooksListCommand(env), newWebhooksGetCommand(env), newWebhooksCreateCommand(env),
		newWebhooksUpdateCommand(env), newWebhooksDeleteCommand(env), newWebhooksRotateCommand(env),
		newWebhooksTestCommand(env), newWebhooksDeliveriesCommand(env), newWebhooksRedeliverCommand(env),
	)
	return cmd
}

func tenantCall(env *Environment, auth *authFlags, cmd *cobra.Command, method, path string, body, out interface{}) error {
	envCtx, err := requireEnvironment(env)
	if err != nil {
		return err
	}
	tc, _, _, err := auth.resolveTenantClient(envCtx, cmd)
	if err != nil {
		return err
	}
	return tc.Call(cmd.Context(), method, path, body, out)
}

func pagePath(base string, limit, offset int) string {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}

func newWebhooksListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var limit, offset int
	var raw bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List webhook subscriptions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var out struct {
				Items      []webhookRow   `json:"items"`
				Pagination map[string]any `json:"pagination"`
			}
			if err := tenantCall(env, &auth, cmd, http.MethodGet, pagePath("/api/webhooks", limit, offset), nil, &out); err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, out)
			}
			if len(out.Items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No webhooks")
				return nil
			}
			rows := make([][]string, 0, len(out.Items))
			for _, w := range out.Items {
				state := "enabled"
				if !w.Enabled {
					state = "paused"
				}
				rows = append(rows, []string{w.ID, w.Name, w.URL, strings.Join(w.Events, ","), state,
					fmt.Sprintf("%d/%d", w.SuccessfulDeliveries, w.TotalDeliveries)})
			}
			renderTable(cmd, []string{"ID", "NAME", "URL", "EVENTS", "STATE", "OK/TOTAL"}, rows)
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size (server maximum 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Rows to skip")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print the raw JSON answer")
	return cmd
}

func newWebhooksGetCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "get <webhook_id>",
		Short: "Show one webhook subscription as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out map[string]any
			if err := tenantCall(env, &auth, cmd, http.MethodGet, "/api/webhooks/"+url.PathEscape(args[0]), nil, &out); err != nil {
				return err
			}
			return printJSON(cmd, out)
		},
	}
	auth.bind(cmd)
	return cmd
}

// headerMap parses repeated KEY=VALUE flags.
func headerMap(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || strings.TrimSpace(k) == "" {
			return nil, fmt.Errorf("--header %q must look like Name=value", p)
		}
		out[strings.TrimSpace(k)] = v
	}
	return out, nil
}

func newWebhooksCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var name, target, description string
	var events, collections, headers []string
	var maxRetries int
	cmd := &cobra.Command{
		Use:   "create --name NAME --url URL --event EVENT [--event EVENT...]",
		Short: "Create a webhook subscription",
		Long: `Create a webhook subscription.

The signing secret is printed ONCE, here. Store it: receivers use it to verify the X-TinyDB-Signature-V2 header, and it is never shown again. If you lose it, run 'tdb tenant webhooks rotate-secret'.`,
		Example: `  tdb tenant webhooks create --name orders --url https://example.com/hook \
    --event document.created --event document.updated --collection orders`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			hdr, err := headerMap(headers)
			if err != nil {
				return err
			}
			body := map[string]any{"name": name, "url": target, "events": events}
			if description != "" {
				body["description"] = description
			}
			if len(collections) > 0 {
				body["collection_ids"] = collections
			}
			if len(hdr) > 0 {
				body["headers"] = hdr
			}
			if cmd.Flags().Changed("max-retries") {
				body["max_retries"] = maxRetries
			}
			var out webhookRow
			if err := tenantCall(env, &auth, cmd, http.MethodPost, "/api/webhooks", body, &out); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created webhook %s (%s)\n", out.Name, out.ID)
			if out.Secret != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Signing secret (shown once, store it now): %s\n", out.Secret)
			}
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().StringVar(&name, "name", "", "Subscription name")
	cmd.Flags().StringVar(&target, "url", "", "Endpoint that receives deliveries")
	cmd.Flags().StringVar(&description, "description", "", "Free-text description")
	cmd.Flags().StringSliceVar(&events, "event", nil, "Event type to subscribe to (repeatable), e.g. document.created")
	cmd.Flags().StringSliceVar(&collections, "collection", nil, "Only these collections (repeatable); default all")
	cmd.Flags().StringSliceVar(&headers, "header", nil, "Extra header sent with each delivery, Name=value (repeatable)")
	cmd.Flags().IntVar(&maxRetries, "max-retries", 5, "Total delivery attempts")
	requireFlag(cmd, "name")
	requireFlag(cmd, "url")
	requireFlag(cmd, "event")
	return cmd
}

func newWebhooksUpdateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var name, target string
	var events []string
	var enabled bool
	cmd := &cobra.Command{
		Use:   "update <webhook_id>",
		Short: "Change a webhook subscription (only the flags you pass)",
		Example: `  tdb tenant webhooks update wh_123 --enabled=false
  tdb tenant webhooks update wh_123 --url https://example.com/new`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("url") {
				body["url"] = target
			}
			if cmd.Flags().Changed("event") {
				body["events"] = events
			}
			if cmd.Flags().Changed("enabled") {
				body["enabled"] = enabled
			}
			if len(body) == 0 {
				return errors.New("nothing to change: pass --name, --url, --event or --enabled")
			}
			var out webhookRow
			if err := tenantCall(env, &auth, cmd, http.MethodPut, "/api/webhooks/"+url.PathEscape(args[0]), body, &out); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated webhook %s\n", out.ID)
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().StringVar(&name, "name", "", "New name")
	cmd.Flags().StringVar(&target, "url", "", "New endpoint")
	cmd.Flags().StringSliceVar(&events, "event", nil, "Replace the subscribed events (repeatable)")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Pause (false) or resume (true) deliveries")
	return cmd
}

func newWebhooksDeleteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "delete <webhook_id>",
		Short: "Delete a webhook subscription and its delivery history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := tenantCall(env, &auth, cmd, http.MethodDelete, "/api/webhooks/"+url.PathEscape(args[0]), nil, nil); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted webhook %s\n", args[0])
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}

func newWebhooksRotateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "rotate-secret <webhook_id>",
		Short: "Issue a new signing secret (shown once); the old one stops working at once",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out webhookRow
			if err := tenantCall(env, &auth, cmd, http.MethodPost, "/api/webhooks/"+url.PathEscape(args[0])+"/rotate-secret", nil, &out); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "New signing secret for %s (shown once, store it now): %s\n", out.ID, out.Secret)
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}

func newWebhooksTestCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "test <webhook_id>",
		Short: "Send a test event now and show the result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out map[string]any
			err := tenantCall(env, &auth, cmd, http.MethodPost, "/api/webhooks/"+url.PathEscape(args[0])+"/test", nil, &out)
			if out != nil {
				_ = printJSON(cmd, out)
			}
			return err
		},
	}
	auth.bind(cmd)
	return cmd
}

func newWebhooksDeliveriesCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var limit, offset int
	var raw bool
	cmd := &cobra.Command{
		Use:   "deliveries <webhook_id>",
		Short: "List recent deliveries of a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out struct {
				Items []deliveryRow `json:"items"`
			}
			if err := tenantCall(env, &auth, cmd, http.MethodGet, pagePath("/api/webhooks/"+url.PathEscape(args[0])+"/deliveries", limit, offset), nil, &out); err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, out)
			}
			if len(out.Items) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No deliveries")
				return nil
			}
			rows := make([][]string, 0, len(out.Items))
			for _, d := range out.Items {
				code := "-"
				if d.StatusCode > 0 {
					code = strconv.Itoa(d.StatusCode)
				}
				rows = append(rows, []string{d.ID, d.EventType, d.Status, strconv.Itoa(d.Attempts), code, d.CreatedAt})
			}
			renderTable(cmd, []string{"ID", "EVENT", "STATUS", "ATTEMPTS", "HTTP", "CREATED"}, rows)
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size (server maximum 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Rows to skip")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print the raw JSON answer")
	return cmd
}

func newWebhooksRedeliverCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "redeliver <delivery_id>",
		Short: "Queue the same event again as a new delivery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out deliveryRow
			if err := tenantCall(env, &auth, cmd, http.MethodPost, "/api/webhooks/deliveries/"+url.PathEscape(args[0])+"/redeliver", nil, &out); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Queued delivery %s\n", out.ID)
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}

// ---- schedules --------------------------------------------------------------

type scheduleRow struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	CollectionID   string `json:"collection_id"`
	CronExpression string `json:"cron_expression"`
	Enabled        bool   `json:"enabled"`
	NextRunAt      string `json:"next_run_at"`
	LastRunAt      string `json:"last_run_at"`
}

type executionRow struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	SnapshotID    string `json:"snapshot_id"`
	DocumentCount int    `json:"document_count"`
	ErrorMessage  string `json:"error"`
	StartedAt     string `json:"started_at"`
}

func newTenantSchedulesCommand(env *Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "schedules",
		Aliases: []string{"schedule"},
		Short:   "Manage backup schedules",
		Long:    "Cron-driven snapshot schedules with retention. Needs a tenant-scoped API key.",
	}
	cmd.AddCommand(
		newSchedulesListCommand(env), newSchedulesCreateCommand(env), newSchedulesSetEnabledCommand(env, true),
		newSchedulesSetEnabledCommand(env, false), newSchedulesDeleteCommand(env),
		newSchedulesTriggerCommand(env), newSchedulesExecutionsCommand(env),
	)
	return cmd
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func newSchedulesListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var limit, offset int
	var raw bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List backup schedules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var out struct {
				Schedules []scheduleRow `json:"schedules"`
				Total     int64         `json:"total"`
			}
			if err := tenantCall(env, &auth, cmd, http.MethodGet, pagePath("/api/schedules", limit, offset), nil, &out); err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, out)
			}
			if len(out.Schedules) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No schedules")
				return nil
			}
			rows := make([][]string, 0, len(out.Schedules))
			for _, s := range out.Schedules {
				state := "enabled"
				if !s.Enabled {
					state = "paused"
				}
				rows = append(rows, []string{s.ID, s.Name, s.CollectionID, s.CronExpression, state, orDash(s.NextRunAt), orDash(s.LastRunAt)})
			}
			renderTable(cmd, []string{"ID", "NAME", "COLLECTION", "CRON", "STATE", "NEXT RUN", "LAST RUN"}, rows)
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size (server maximum 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Rows to skip")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print the raw JSON answer")
	return cmd
}

func newSchedulesCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var name, collectionID, cron, description string
	var keepLast, keepDays, keepDaily, keepWeekly, keepMonthly int
	var paused bool
	cmd := &cobra.Command{
		Use:   "create --collection-id ID --name NAME --cron EXPR",
		Short: "Create a backup schedule",
		Long: `Create a cron-driven snapshot schedule.

--cron is the standard five-field syntax (minute hour day month weekday). Retention only ever deletes snapshots this schedule created, and is skipped when a run fails. Combine --keep-daily/--keep-weekly/--keep-monthly for grandfather-father-son retention.`,
		Example: `  tdb tenant schedules create --collection-id b1f2... --name nightly --cron "0 3 * * *" --keep-daily 7 --keep-weekly 4`,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			retention := map[string]int{}
			for k, v := range map[string]int{"keep_last": keepLast, "keep_days": keepDays, "keep_daily": keepDaily, "keep_weekly": keepWeekly, "keep_monthly": keepMonthly} {
				if v > 0 {
					retention[k] = v
				}
			}
			body := map[string]any{"collection_id": collectionID, "name": name, "cron_expression": cron, "enabled": !paused}
			if description != "" {
				body["description"] = description
			}
			if len(retention) > 0 {
				body["retention_policy"] = retention
			}
			var out scheduleRow
			if err := tenantCall(env, &auth, cmd, http.MethodPost, "/api/schedules", body, &out); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created schedule %s (%s), next run %s\n", out.Name, out.ID, orDash(out.NextRunAt))
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().StringVar(&collectionID, "collection-id", "", "Collection to snapshot")
	cmd.Flags().StringVar(&name, "name", "", "Schedule name")
	cmd.Flags().StringVar(&cron, "cron", "", `Five-field cron expression, e.g. "0 3 * * *"`)
	cmd.Flags().StringVar(&description, "description", "", "Free-text description")
	cmd.Flags().IntVar(&keepLast, "keep-last", 0, "Keep this many newest snapshots")
	cmd.Flags().IntVar(&keepDays, "keep-days", 0, "Keep snapshots newer than this many days")
	cmd.Flags().IntVar(&keepDaily, "keep-daily", 0, "Keep the newest snapshot of each of this many days")
	cmd.Flags().IntVar(&keepWeekly, "keep-weekly", 0, "Keep the newest snapshot of each of this many weeks")
	cmd.Flags().IntVar(&keepMonthly, "keep-monthly", 0, "Keep the newest snapshot of each of this many months")
	cmd.Flags().BoolVar(&paused, "paused", false, "Create it disabled")
	requireFlag(cmd, "collection-id")
	requireFlag(cmd, "name")
	requireFlag(cmd, "cron")
	return cmd
}

func newSchedulesSetEnabledCommand(env *Environment, enable bool) *cobra.Command {
	var auth authFlags
	use, short := "disable <schedule_id>", "Pause a schedule (it stops running; nothing is deleted)"
	if enable {
		use, short = "enable <schedule_id>", "Resume a paused schedule"
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out scheduleRow
			if err := tenantCall(env, &auth, cmd, http.MethodPut, "/api/schedules/"+url.PathEscape(args[0]), map[string]any{"enabled": enable}, &out); err != nil {
				return err
			}
			state := "paused"
			if out.Enabled {
				state = "enabled"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Schedule %s is %s\n", args[0], state)
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}

func newSchedulesDeleteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "delete <schedule_id>",
		Short: "Delete a schedule (its snapshots are kept)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := tenantCall(env, &auth, cmd, http.MethodDelete, "/api/schedules/"+url.PathEscape(args[0]), nil, nil); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted schedule %s\n", args[0])
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}

func newSchedulesTriggerCommand(env *Environment) *cobra.Command {
	var auth authFlags
	cmd := &cobra.Command{
		Use:   "trigger <schedule_id>",
		Short: "Run a schedule now",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out executionRow
			if err := tenantCall(env, &auth, cmd, http.MethodPost, "/api/schedules/"+url.PathEscape(args[0])+"/trigger", nil, &out); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Run %s: %s", out.ID, out.Status)
			if out.SnapshotID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), " (snapshot %s, %d documents)", out.SnapshotID, out.DocumentCount)
			}
			fmt.Fprintln(cmd.OutOrStdout())
			if out.ErrorMessage != "" {
				return fmt.Errorf("backup failed: %s", out.ErrorMessage)
			}
			return nil
		},
	}
	auth.bind(cmd)
	return cmd
}

func newSchedulesExecutionsCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var limit, offset int
	var raw bool
	cmd := &cobra.Command{
		Use:   "executions <schedule_id>",
		Short: "Show past runs of a schedule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out struct {
				Executions []executionRow `json:"executions"`
			}
			if err := tenantCall(env, &auth, cmd, http.MethodGet, pagePath("/api/schedules/"+url.PathEscape(args[0])+"/executions", limit, offset), nil, &out); err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, out)
			}
			if len(out.Executions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No runs yet")
				return nil
			}
			sort.SliceStable(out.Executions, func(i, j int) bool { return out.Executions[i].StartedAt > out.Executions[j].StartedAt })
			rows := make([][]string, 0, len(out.Executions))
			for _, e := range out.Executions {
				detail := orDash(e.SnapshotID)
				if e.ErrorMessage != "" {
					detail = e.ErrorMessage
				}
				rows = append(rows, []string{e.ID, e.Status, strconv.Itoa(e.DocumentCount), detail, e.StartedAt})
			}
			renderTable(cmd, []string{"ID", "STATUS", "DOCS", "SNAPSHOT / ERROR", "STARTED"}, rows)
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().IntVar(&limit, "limit", 0, "Page size (server maximum 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Rows to skip")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print the raw JSON answer")
	return cmd
}
