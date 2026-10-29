package cli

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
)

// requireFlag marks a flag required. The flag is defined a line earlier, so a
// failure here is a programming error and must not be ignored.
func requireFlag(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(fmt.Sprintf("tdb: required flag %q is not defined: %v", name, err))
	}
}

func newTenantUsageCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var days int
	var raw bool

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show plan limits and current usage",
		Long: `Show where the tenant stands against its plan: effective limits, current usage, and daily request counts.

Needs a tenant-scoped key; application-scoped keys cannot read tenant totals. The figures are measured with a scan of the tenant's documents, so run it on demand, not in a tight loop.`,
		Example: `  tdb tenant usage --api-key $API_KEY
  tdb tenant usage --days 7 --raw`,
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			u, err := tenantClient.Usage(cmd.Context(), days)
			if err != nil {
				return err
			}
			if raw {
				return printJSON(cmd, u)
			}
			out := cmd.OutOrStdout()
			plan := u.Plan
			if plan == "" {
				plan = "-"
			}
			fmt.Fprintf(out, "Plan:         %s\n", plan)
			if u.Suspended {
				fmt.Fprintf(out, "SUSPENDED:    %s\n", u.SuspendedReason)
			}
			rows := [][]string{
				{"documents", fmt.Sprint(u.Usage.Documents), limitLabel(u.Limits.Documents, false)},
				{"storage", formatBytes(u.Usage.StorageBytes), limitLabel(u.Limits.StorageBytes, true)},
				{"collections", fmt.Sprint(u.Collections), limitLabel(u.Limits.Collections, false)},
				{"applications", fmt.Sprint(u.Applications), limitLabel(u.Limits.Applications, false)},
				{"api keys", fmt.Sprint(u.APIKeys), limitLabel(u.Limits.APIKeys, false)},
			}
			if len(u.Requests) > 0 {
				today := u.Requests[len(u.Requests)-1]
				rows = append(rows, []string{"requests (" + today.Day + ")", fmt.Sprint(today.Requests), limitLabel(u.Limits.RequestsPerDay, false)})
			}
			renderTable(cmd, []string{"RESOURCE", "USED", "LIMIT"}, rows)
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().IntVar(&days, "days", 30, "Days of request history to include")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	return cmd
}

func limitLabel(v int64, bytes bool) string {
	if v <= 0 {
		return "unlimited"
	}
	if bytes {
		return formatBytes(v)
	}
	return fmt.Sprint(v)
}

func newTenantAuditVerifyCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var raw bool
	var expectHash string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify the audit log's tamper-evident hash chain",
		Long: `Walk the tenant's audit hash chain and report the first entry that was altered or whose neighbours were removed.

Exits non-zero when the chain is broken. Entries written in the last few seconds are 'unsealed' and not yet covered. Record last_hash somewhere outside the database and pass it later with --expect-hash to detect that entries were truncated.`,
		Example: `  tdb tenant audit verify --api-key $API_KEY
  tdb tenant audit verify --expect-hash 9f2c... --api-key $API_KEY`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			v, err := tenantClient.VerifyAudit(cmd.Context())
			if err != nil {
				return err
			}
			if raw {
				if err := printJSON(cmd, v); err != nil {
					return err
				}
			} else {
				out := cmd.OutOrStdout()
				if v.OK {
					fmt.Fprintf(out, "OK: %d entries verified (seq %d..%d), %d not yet sealed\n", v.Checked, v.FirstSeq, v.LastSeq, v.Unsealed)
					if v.LastHash != "" {
						fmt.Fprintf(out, "last_hash: %s\n", v.LastHash)
					}
				} else {
					fmt.Fprintf(out, "BROKEN at seq %d: %s\n", v.BrokenSeq, v.Reason)
				}
			}
			if !v.OK {
				return errors.New("audit chain verification failed")
			}
			if want := strings.TrimSpace(expectHash); want != "" && !strings.EqualFold(want, v.LastHash) {
				return fmt.Errorf("last_hash %s does not match the expected %s: entries may have been removed or rewritten", v.LastHash, want)
			}
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	cmd.Flags().StringVar(&expectHash, "expect-hash", "", "last_hash recorded from an earlier run; fail if the chain no longer ends there")
	return cmd
}

func newTenantSnapshotsVerifyCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var snapshotID string
	var raw bool

	cmd := &cobra.Command{
		Use:   "verify --snapshot SNAPSHOT_ID",
		Short: "Check a snapshot's integrity without restoring it",
		Long: `Dry-run a restore: verify the snapshot's checksum, that its payload decodes, belongs to the collection and holds the recorded number of documents, and show how many live documents a replace_all restore would delete.

Exits non-zero when the snapshot is not valid.`,
		Example: `  tdb tenant snapshots verify --snapshot snap-123 --api-key $API_KEY`,
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			v, err := tenantClient.VerifySnapshot(cmd.Context(), strings.TrimSpace(snapshotID))
			if err != nil {
				return err
			}
			if raw {
				if err := printJSON(cmd, v); err != nil {
					return err
				}
			} else {
				out := cmd.OutOrStdout()
				verdict := "VALID"
				if !v.Valid {
					verdict = "INVALID"
				}
				fmt.Fprintf(out, "%s snapshot %s\n", verdict, v.SnapshotID)
				fmt.Fprintf(out, "  documents: %d recorded, %d in payload\n", v.DocumentsRecorded, v.DocumentsInPayload)
				switch {
				case !v.ChecksumRecorded:
					fmt.Fprintln(out, "  checksum:  not recorded (snapshot predates integrity checks)")
				case v.ChecksumMatches != nil && *v.ChecksumMatches:
					fmt.Fprintln(out, "  checksum:  matches")
				default:
					fmt.Fprintln(out, "  checksum:  MISMATCH")
				}
				fmt.Fprintf(out, "  a replace_all restore would delete %d live documents\n", v.ExistingDocuments)
				for _, p := range v.Problems {
					fmt.Fprintf(out, "  problem: %s\n", p)
				}
			}
			if !v.Valid {
				return errors.New("snapshot failed verification")
			}
			return nil
		},
	}
	auth.bind(cmd)
	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Snapshot ID (required)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")
	requireFlag(cmd, "snapshot")
	return cmd
}

func newTenantWatchCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var since uint64
	var raw bool

	cmd := &cobra.Command{
		Use:   "watch <collection>",
		Short: "Stream changes to a collection, resuming after a disconnect",
		Long: `Print each change to a collection as it happens (create, update, delete).

Every change carries a seq. With --since SEQ the server first replays what was recorded after that point, then continues live; if the connection drops, watch reconnects and resumes from the last seq it printed, so nothing is missed. A 'lagged' line means this consumer was too slow; the missed changes follow it.

Stop with Ctrl+C.`,
		Example: `  tdb tenant watch orders --api-key $API_KEY
  tdb tenant watch orders --since 4812 --raw --api-key $API_KEY | jq .id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}
			collection := strings.TrimSpace(args[0])
			if collection == "" {
				return errors.New("collection name cannot be empty")
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			var sincePtr *uint64
			if cmd.Flags().Changed("since") {
				sincePtr = &since
			}
			return tenantClient.Watch(ctx, collection, auth.appID, sincePtr, time.Second, func(ev clientpkg.ChangeEvent) error {
				return printChange(cmd, ev, raw)
			})
		},
	}
	auth.bindWithApp(cmd)
	cmd.Flags().Uint64Var(&since, "since", 0, "Resume after this seq (the last one you processed)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print one JSON object per line")
	return cmd
}

func printChange(cmd *cobra.Command, ev clientpkg.ChangeEvent, raw bool) error {
	if raw {
		return printCompactJSON(cmd, ev)
	}
	seq := "-"
	if ev.Seq > 0 {
		seq = fmt.Sprint(ev.Seq)
	}
	id := ev.ID
	if id == "" {
		id = "-"
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s  seq=%s  %-7s  %s\n", ev.Timestamp.Format(time.RFC3339), seq, strings.ToUpper(ev.Type), id)
	return err
}
