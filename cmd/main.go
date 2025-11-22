package main

import (
	"context"
	"fmt"
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
		fmt.Println("Loading AWS configuration...")

		// Get flags
		awsProfile, _ := cmd.Flags().GetString("aws-profile")
		region, _ := cmd.Flags().GetString("region")
		bucket, _ := cmd.Flags().GetString("bucket")
		vaultPath, _ := cmd.Flags().GetString("vault-path")
		createTar, _ := cmd.Flags().GetBool("tar")
		deleteLocal, _ := cmd.Flags().GetBool("delete")
		tarDir, _ := cmd.Flags().GetString("tar-dir")
		tarKeep, _ := cmd.Flags().GetInt("tar-keep")

		// Create context with 30 second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Load AWS configuration with optional profile and region
		var cfg aws.Config
		var err error
		if awsProfile != "" {
			fmt.Printf("Using AWS profile: %s\n", awsProfile)
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
			fmt.Fprintf(os.Stderr, "Failed to load AWS configuration: %v\n", err)
			os.Exit(1)
		}

		// Create S3 client
		s3Client := s3.NewFromConfig(cfg)
		fmt.Printf("Successfully connected to AWS in region: %s\n", cfg.Region)

		// Ensure vaultPath doesn't start with / and ends with /
		vaultPath = strings.TrimPrefix(vaultPath, "/")
		if !strings.HasSuffix(vaultPath, "/") {
			vaultPath += "/"
		}

		fmt.Printf("Syncing vault from s3://%s/%s\n", bucket, vaultPath)

		// Create local vault directory
		localDir := "./vault"
		if err := os.MkdirAll(localDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create local directory: %v\n", err)
			os.Exit(1)
		}

		// Sync vault from S3
		stats, err := s3sync.SyncVaultFromS3(ctx, s3Client, bucket, vaultPath, localDir, deleteLocal)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync vault: %v\n", err)
			os.Exit(1)
		}

		// Print sync statistics
		fmt.Printf("\n✓ Vault sync completed successfully to %s\n", localDir)
		fmt.Printf("  Downloaded: %d files\n", stats.Downloaded)
		fmt.Printf("  Skipped (up-to-date): %d files\n", stats.Skipped)
		if stats.Deleted > 0 {
			fmt.Printf("  Deleted: %d files\n", stats.Deleted)
		}
		if stats.Failed > 0 {
			fmt.Printf("  Failed: %d files\n", stats.Failed)
		}

		// Create tarball if requested
		if createTar {
			vaultName := filepath.Base(strings.TrimSuffix(vaultPath, "/"))

			// Create the tarball
			outputDir, err := tar.CreateVaultTarball(localDir, vaultName, tarDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create tarball: %v\n", err)
				os.Exit(1)
			}

			// Clean up old tarballs if tar-keep is set
			if err := tar.CleanupOldTarballs(outputDir, vaultName, tarKeep); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to cleanup old tarballs: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("aws-profile", "p", "", "AWS profile to use from ~/.aws/credentials")
	runCmd.Flags().StringP("region", "r", "us-east-1", "AWS region")
	runCmd.Flags().StringP("bucket", "b", "", "AWS S3 bucket fecth vault from")
	runCmd.Flags().StringP("vault-path", "v", "", "Path to the Obsidian vault in the S3 bucket")
	runCmd.Flags().BoolP("tar", "t", false, "Create a tar.gz backup of the vault with auto-generated filename")
	runCmd.Flags().BoolP("delete", "d", false, "Delete local files that don't exist in S3")
	runCmd.Flags().String("tar-dir", "", "Directory to save tarballs (default: current directory)")
	runCmd.Flags().Int("tar-keep", 5, "Number of tarballs to keep (deletes oldest, 0 = keep all)")

	// Mark required flags
	runCmd.MarkFlagRequired("bucket")
	runCmd.MarkFlagRequired("vault-path")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
