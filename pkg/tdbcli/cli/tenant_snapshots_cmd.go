package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	clientpkg "github.com/cubetiqlabs/tdb-cli/pkg/tdbcli/client"
)

func newTenantSnapshotsCommand(env *Environment) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshots",
		Aliases: []string{"snapshot", "backup", "backups"},
		Short:   "Manage collection snapshots (backups)",
		Long:    "Create, restore, list, and delete snapshots for collections",
	}

	cmd.AddCommand(newTenantSnapshotsListCommand(env))
	cmd.AddCommand(newTenantSnapshotsCreateCommand(env))
	cmd.AddCommand(newTenantSnapshotsRestoreCommand(env))
	cmd.AddCommand(newTenantSnapshotsDeleteCommand(env))
	cmd.AddCommand(newTenantSnapshotsGetCommand(env))

	return cmd
}

// newTenantSnapshotsListCommand lists all snapshots for a tenant
func newTenantSnapshotsListCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var collectionID string
	var limit int
	var offset int
	var raw bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snapshots",
		Long:  "List all snapshots for the tenant, optionally filtered by collection",
		Example: `  # List all snapshots
  tdb tenant snapshots list --api-key $API_KEY

  # List snapshots for a specific collection
  tdb tenant snapshots list --api-key $API_KEY --collection my-collection

  # List with pagination
  tdb tenant snapshots list --api-key $API_KEY --limit 10 --offset 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}

			snapshots, err := tenantClient.ListSnapshots(cmd.Context(), collectionID, limit, offset)
			if err != nil {
				return fmt.Errorf("failed to list snapshots: %w", err)
			}

			if raw {
				return printJSON(cmd, snapshots)
			}

			if len(snapshots) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No snapshots found")
				return nil
			}

			rows := make([][]string, 0, len(snapshots))
			for _, snap := range snapshots {
				snapshotType := "full"
				if snap.SnapshotType != "" {
					snapshotType = snap.SnapshotType
				}

				encrypted := "no"
				if snap.Encrypted {
					encrypted = "yes"
				}

				storage := "local"
				if snap.StorageProvider != "" {
					storage = snap.StorageProvider
				}

				rows = append(rows, []string{
					snap.ID,
					snap.CollectionName,
					snap.Name,
					snapshotType,
					fmt.Sprintf("%d", snap.DocumentCount),
					formatBytes(snap.SizeBytes),
					encrypted,
					storage,
					formatTime(snap.CreatedAt),
				})
			}

			renderTable(cmd, []string{
				"ID", "COLLECTION", "NAME", "TYPE", "DOCS", "SIZE", "ENCRYPTED", "STORAGE", "CREATED",
			}, rows)

			return nil
		},
	}

	auth.bind(cmd)
	cmd.Flags().StringVar(&collectionID, "collection", "", "Filter by collection ID")
	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of snapshots to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Number of snapshots to skip")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	return cmd
}

// newTenantSnapshotsCreateCommand creates a new snapshot
func newTenantSnapshotsCreateCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var collectionID string
	var name string
	var description string
	var incremental bool
	var parentSnapshotID string
	var encrypt bool
	var storageProvider string
	var raw bool

	cmd := &cobra.Command{
		Use:   "create --collection COLLECTION_ID --name NAME",
		Short: "Create a new snapshot",
		Long:  "Create a full or incremental snapshot of a collection",
		Example: `  # Create a full snapshot
  tdb tenant snapshots create --api-key $API_KEY --collection my-coll --name "Daily backup"

  # Create an encrypted snapshot with S3 storage
  tdb tenant snapshots create --api-key $API_KEY --collection my-coll --name "Production backup" \
    --encrypt --storage s3

  # Create an incremental snapshot
  tdb tenant snapshots create --api-key $API_KEY --collection my-coll --name "Incremental" \
    --incremental --parent-snapshot parent-id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if collectionID == "" {
				return fmt.Errorf("--collection is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}

			req := clientpkg.CreateSnapshotRequest{
				CollectionID:     collectionID,
				Name:             name,
				Description:      description,
				Incremental:      incremental,
				ParentSnapshotID: parentSnapshotID,
				Encrypt:          encrypt,
				StorageProvider:  storageProvider,
			}

			snapshot, err := tenantClient.CreateSnapshot(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("failed to create snapshot: %w", err)
			}

			if raw {
				return printJSON(cmd, snapshot)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Snapshot created successfully\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  ID:          %s\n", snapshot.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Collection:  %s\n", snapshot.CollectionName)
			fmt.Fprintf(cmd.OutOrStdout(), "  Name:        %s\n", snapshot.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "  Type:        %s\n", snapshot.SnapshotType)
			fmt.Fprintf(cmd.OutOrStdout(), "  Documents:   %d\n", snapshot.DocumentCount)
			fmt.Fprintf(cmd.OutOrStdout(), "  Size:        %s\n", formatBytes(snapshot.SizeBytes))
			if snapshot.Encrypted {
				fmt.Fprintf(cmd.OutOrStdout(), "  Encrypted:   yes\n")
			}
			if snapshot.StorageProvider != "" && snapshot.StorageProvider != "local" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Storage:     %s\n", snapshot.StorageProvider)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  Created:     %s\n", formatTime(snapshot.CreatedAt))

			return nil
		},
	}

	auth.bind(cmd)
	cmd.Flags().StringVar(&collectionID, "collection", "", "Collection ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Snapshot name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Snapshot description")
	cmd.Flags().BoolVar(&incremental, "incremental", false, "Create incremental snapshot")
	cmd.Flags().StringVar(&parentSnapshotID, "parent-snapshot", "", "Parent snapshot ID for incremental snapshots")
	cmd.Flags().BoolVar(&encrypt, "encrypt", false, "Encrypt snapshot data")
	cmd.Flags().StringVar(&storageProvider, "storage", "", "Storage provider (local, s3, gcs)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	cmd.MarkFlagRequired("collection")
	cmd.MarkFlagRequired("name")

	return cmd
}

// newTenantSnapshotsRestoreCommand restores a snapshot
func newTenantSnapshotsRestoreCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var snapshotID string
	var targetCollectionID string
	var raw bool

	cmd := &cobra.Command{
		Use:   "restore --snapshot SNAPSHOT_ID [--target-collection COLLECTION_ID]",
		Short: "Restore a snapshot",
		Long:  "Restore a snapshot to its original collection or a different collection",
		Example: `  # Restore to original collection
  tdb tenant snapshots restore --api-key $API_KEY --snapshot snap-123

  # Restore to a different collection
  tdb tenant snapshots restore --api-key $API_KEY --snapshot snap-123 --target-collection new-coll`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if snapshotID == "" {
				return fmt.Errorf("--snapshot is required")
			}

			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}

			req := clientpkg.RestoreSnapshotRequest{
				TargetCollectionID: targetCollectionID,
			}

			result, err := tenantClient.RestoreSnapshot(cmd.Context(), snapshotID, req)
			if err != nil {
				return fmt.Errorf("failed to restore snapshot: %w", err)
			}

			if raw {
				return printJSON(cmd, result)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Snapshot restored successfully\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  Collection:       %s\n", result.CollectionID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Documents restored: %d\n", result.DocumentsRestored)

			return nil
		},
	}

	auth.bind(cmd)
	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Snapshot ID (required)")
	cmd.Flags().StringVar(&targetCollectionID, "target-collection", "", "Target collection ID (defaults to original)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	cmd.MarkFlagRequired("snapshot")

	return cmd
}

// newTenantSnapshotsDeleteCommand deletes a snapshot
func newTenantSnapshotsDeleteCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var snapshotID string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete --snapshot SNAPSHOT_ID",
		Short: "Delete a snapshot",
		Long:  "Permanently delete a snapshot",
		Example: `  # Delete a snapshot
  tdb tenant snapshots delete --api-key $API_KEY --snapshot snap-123 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if snapshotID == "" {
				return fmt.Errorf("--snapshot is required")
			}

			if !force {
				return fmt.Errorf("use --force to confirm deletion")
			}

			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}

			err = tenantClient.DeleteSnapshot(cmd.Context(), snapshotID)
			if err != nil {
				return fmt.Errorf("failed to delete snapshot: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Snapshot %s deleted successfully\n", snapshotID)

			return nil
		},
	}

	auth.bind(cmd)
	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Snapshot ID (required)")
	cmd.Flags().BoolVar(&force, "force", false, "Force deletion without confirmation")

	cmd.MarkFlagRequired("snapshot")

	return cmd
}

// newTenantSnapshotsGetCommand gets details of a snapshot
func newTenantSnapshotsGetCommand(env *Environment) *cobra.Command {
	var auth authFlags
	var snapshotID string
	var raw bool

	cmd := &cobra.Command{
		Use:   "get --snapshot SNAPSHOT_ID",
		Short: "Get snapshot details",
		Long:  "Retrieve detailed information about a specific snapshot",
		Example: `  # Get snapshot details
  tdb tenant snapshots get --api-key $API_KEY --snapshot snap-123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if snapshotID == "" {
				return fmt.Errorf("--snapshot is required")
			}

			envCtx, err := requireEnvironment(env)
			if err != nil {
				return err
			}
			tenantClient, _, _, err := auth.resolveTenantClient(envCtx, cmd)
			if err != nil {
				return err
			}

			snapshot, err := tenantClient.GetSnapshot(cmd.Context(), snapshotID)
			if err != nil {
				return fmt.Errorf("failed to get snapshot: %w", err)
			}

			if raw {
				return printJSON(cmd, snapshot)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Snapshot Details\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  ID:               %s\n", snapshot.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Collection:       %s (%s)\n", snapshot.CollectionName, snapshot.CollectionID)
			fmt.Fprintf(cmd.OutOrStdout(), "  Name:             %s\n", snapshot.Name)
			if snapshot.Description != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Description:      %s\n", snapshot.Description)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  Type:             %s\n", snapshot.SnapshotType)
			if snapshot.ParentSnapshotID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Parent Snapshot:  %s\n", snapshot.ParentSnapshotID)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  Documents:        %d\n", snapshot.DocumentCount)
			fmt.Fprintf(cmd.OutOrStdout(), "  Size:             %s\n", formatBytes(snapshot.SizeBytes))
			fmt.Fprintf(cmd.OutOrStdout(), "  Compressed:       %v\n", snapshot.Compressed)
			fmt.Fprintf(cmd.OutOrStdout(), "  Encrypted:        %v\n", snapshot.Encrypted)
			if snapshot.EncryptionKeyID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Encryption Key:   %s\n", snapshot.EncryptionKeyID)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  Storage Provider: %s\n", snapshot.StorageProvider)
			if snapshot.StoragePath != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Storage Path:     %s\n", snapshot.StoragePath)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  Created:          %s\n", formatTime(snapshot.CreatedAt))
			if snapshot.CreatedBy != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Created By:       %s\n", snapshot.CreatedBy)
			}
			if snapshot.ExpiresAt != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  Expires:          %s\n", formatTime(*snapshot.ExpiresAt))
			}

			return nil
		},
	}

	auth.bind(cmd)
	cmd.Flags().StringVar(&snapshotID, "snapshot", "", "Snapshot ID (required)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw JSON response")

	cmd.MarkFlagRequired("snapshot")

	return cmd
}
