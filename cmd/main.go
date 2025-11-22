package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vault-sync",
	Short: "Obsidian S3 Vault Sync",
	Long:  `A CLI tool to sync Obsidian vaults from S3 locally and manage tar backups.`,
}

// createTarball creates a tar.gz archive of the specified directory
func createTarball(sourceDir, targetPath string) error {
	// Create the tar.gz file
	tarFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer tarFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through the directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Update header name to be relative to source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If it's a file (not a directory), write its contents
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file to tar: %w", err)
			}
		}

		return nil
	})
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

		// List objects in the S3 bucket
		listResp, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket: &bucket,
			Prefix: &vaultPath,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to list objects: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d objects to download\n", len(listResp.Contents))

		// Download each object
		for _, obj := range listResp.Contents {
			// Skip if it's a directory marker
			if strings.HasSuffix(*obj.Key, "/") {
				continue
			}

			// Get relative path by removing the vault path prefix
			relPath := strings.TrimPrefix(*obj.Key, vaultPath)
			localPath := filepath.Join(localDir, relPath)

			// Create parent directories if needed
			if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create directory for %s: %v\n", localPath, err)
				continue
			}

			// Download the file
			fmt.Printf("Downloading %s -> %s\n", *obj.Key, localPath)
			result, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: &bucket,
				Key:    obj.Key,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to download %s: %v\n", *obj.Key, err)
				continue
			}

			// Create local file
			file, err := os.Create(localPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create file %s: %v\n", localPath, err)
				result.Body.Close()
				continue
			}

			// Copy S3 object to local file
			_, err = io.Copy(file, result.Body)
			result.Body.Close()
			file.Close()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write file %s: %v\n", localPath, err)
				continue
			}
		}

		fmt.Printf("✓ Vault sync completed successfully to %s\n", localDir)

		// Create tarball if requested
		if createTar {
			// Generate tarball filename from vault path and timestamp
			vaultName := filepath.Base(strings.TrimSuffix(vaultPath, "/"))
			timestamp := time.Now().Format("2006-01-02_15-04-05")
			tarOutput := fmt.Sprintf("%s-%s.tar.gz", vaultName, timestamp)

			fmt.Printf("Creating tarball at %s...\n", tarOutput)
			if err := createTarball(localDir, tarOutput); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create tarball: %v\n", err)
				os.Exit(1)
			}

			// Get tarball size
			tarInfo, err := os.Stat(tarOutput)
			if err == nil {
				sizeMB := float64(tarInfo.Size()) / (1024 * 1024)
				fmt.Printf("✓ Tarball created successfully: %s (%.2f MB)\n", tarOutput, sizeMB)
			} else {
				fmt.Printf("✓ Tarball created successfully: %s\n", tarOutput)
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
