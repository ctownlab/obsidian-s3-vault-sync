package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ctownlab/obsidian-s3-vault-sync/internal/s3sync"
	"github.com/ctownlab/obsidian-s3-vault-sync/internal/tar"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vault-sync",
	Short: "Obsidian S3 Vault Sync",
	Long:  `A CLI tool to sync Obsidian vaults from S3 locally and manage tar backups.`,
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the vault sync",
	Long:  `Sync your Obsidian vault from S3 to your local directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get flags
		awsProfile, _ := cmd.Flags().GetString("aws-profile")
		region, _ := cmd.Flags().GetString("region")
		bucket, _ := cmd.Flags().GetString("bucket")
		vaultPaths, _ := cmd.Flags().GetStringSlice("vault-path")
		createTar, _ := cmd.Flags().GetBool("tar")
		deleteLocal, _ := cmd.Flags().GetBool("delete")
		vaultDir, _ := cmd.Flags().GetString("vault-dir")
		tarDir, _ := cmd.Flags().GetString("tar-dir")
		tarKeep, _ := cmd.Flags().GetInt("tar-keep")
		logLevel, _ := cmd.Flags().GetString("log-level")

		// Setup logger
		var level slog.Level
		switch strings.ToLower(logLevel) {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
		slog.SetDefault(logger)

		slog.Info("Loading AWS configuration")

		slog.Debug("DEBUG MODE ENABLED")

		// Create context with 30 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Load AWS configuration with optional profile and region
		var cfg aws.Config
		var err error
		if awsProfile != "" {
			slog.Info("Using AWS profile", "profile", awsProfile)
			cfg, err = config.LoadDefaultConfig(ctx,
				config.WithSharedConfigProfile(awsProfile),
				config.WithRegion(region),
			)
		} else {
			cfg, err = config.LoadDefaultConfig(ctx,
				config.WithRegion(region),
			)
		}

		if err != nil {
			slog.Error("Failed to load AWS configuration", "error", err)
			os.Exit(1)
		}

		// Create S3 client
		s3Client := s3.NewFromConfig(cfg)
		slog.Info("Successfully connected to AWS", "region", cfg.Region)

		slog.Info("Starting vault sync", "vault_count", len(vaultPaths))

		// Sync each vault
		for i, vaultPath := range vaultPaths {
			slog.Info("Processing vault",
				"vault", vaultPath,
				"progress", i+1,
				"total", len(vaultPaths))

			// Ensure vaultPath doesn't start with / and ends with /
			vaultPath = strings.TrimPrefix(vaultPath, "/")
			if !strings.HasSuffix(vaultPath, "/") {
				vaultPath += "/"
			}

			slog.Info("Syncing vault from S3",
				"bucket", bucket,
				"path", vaultPath)

			// Get vault name for directory
			vaultName := filepath.Base(strings.TrimSuffix(vaultPath, "/"))

			// Create local vault directory (each vault gets its own subdirectory)
			baseVaultDir := vaultDir
			if baseVaultDir == "" {
				baseVaultDir = "./vault"
			}
			localDir := filepath.Join(baseVaultDir, vaultName)
			if err := os.MkdirAll(localDir, 0755); err != nil {
				slog.Error("Failed to create local directory",
					"directory", localDir,
					"error", err)
				continue
			}

			// Sync vault from S3
			stats, err := s3sync.SyncVaultFromS3(ctx, s3Client, bucket, vaultPath, localDir, deleteLocal)
			if err != nil {
				slog.Error("Failed to sync vault",
					"vault", vaultPath,
					"error", err)
				continue
			}

			// Log sync statistics
			slog.Info("Vault sync completed",
				"vault", vaultName,
				"directory", localDir,
				"downloaded", stats.Downloaded,
				"skipped", stats.Skipped,
				"deleted", stats.Deleted,
				"failed", stats.Failed)

			// Create tarball if requested
			if createTar {
				slog.Info("Creating tarball", "vault", vaultName)

				// Determine tarball directory (vault-specific subdirectory)
				var vaultTarDir string
				if tarDir != "" {
					vaultTarDir = filepath.Join(tarDir, vaultName)
				} else {
					vaultTarDir = vaultName
				}

				outputDir, err := tar.CreateVaultTarball(localDir, vaultName, vaultTarDir)
				if err != nil {
					slog.Error("Failed to create tarball",
						"vault", vaultName,
						"error", err)
					continue
				}

				// Clean up old tarballs if tar-keep is set
				if err := tar.CleanupOldTarballs(outputDir, vaultName, tarKeep); err != nil {
					slog.Warn("Failed to cleanup old tarballs",
						"vault", vaultName,
						"error", err)
				}
			}
		}

		slog.Info("All vaults processed successfully")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("aws-profile", "p", "", "AWS profile to use from ~/.aws/credentials")
	runCmd.Flags().StringP("region", "r", "us-east-1", "AWS region")
	runCmd.Flags().StringP("bucket", "b", "", "AWS S3 bucket fecth vault from")
	runCmd.Flags().StringSliceP("vault-path", "v", []string{}, "Path(s) to Obsidian vault(s) in S3 (can specify multiple times)")
	runCmd.Flags().BoolP("tar", "t", false, "Create a tar.gz backup of the vault with auto-generated filename")
	runCmd.Flags().BoolP("delete", "d", false, "Delete local files that don't exist in S3")
	runCmd.Flags().String("vault-dir", "", "Directory to save synced vaults (default: ./vault)")
	runCmd.Flags().String("tar-dir", "", "Directory to save tarballs (default: current directory)")
	runCmd.Flags().Int("tar-keep", 5, "Number of tarballs to keep (deletes oldest, 0 = keep all)")
	runCmd.Flags().String("log-level", "info", "Log level (debug, info, warn, error)")

	// Mark required flags
	runCmd.MarkFlagRequired("bucket")
	runCmd.MarkFlagRequired("vault-path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		slog.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}
