package s3sync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Stats holds statistics from an S3 sync operation
type Stats struct {
	Downloaded int
	Skipped    int
	Deleted    int
	Failed     int
}

// SyncVaultFromS3 syncs files from S3 to local directory
func SyncVaultFromS3(ctx context.Context, s3Client *s3.Client, bucket, vaultPath, localDir string, deleteLocal bool) (*Stats, error) {
	stats := &Stats{}

	// List objects in the S3 bucket
	listResp, err := s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &vaultPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	fmt.Printf("Found %d objects in S3\n", len(listResp.Contents))

	// Track all S3 files (relative paths) for deletion check
	s3Files := make(map[string]bool)

	// Download each object
	for _, obj := range listResp.Contents {
		// Skip if it's a directory marker
		if strings.HasSuffix(*obj.Key, "/") {
			continue
		}

		// Get relative path by removing the vault path prefix
		relPath := strings.TrimPrefix(*obj.Key, vaultPath)
		localPath := filepath.Join(localDir, relPath)

		// Track this S3 file for deletion check later
		s3Files[relPath] = true

		// Check if local file exists and compare
		needsDownload := true
		if localInfo, err := os.Stat(localPath); err == nil {
			// File exists, compare size and modification time
			localSize := localInfo.Size()
			s3Size := *obj.Size
			s3ModTime := *obj.LastModified

			if localSize == s3Size && !s3ModTime.After(localInfo.ModTime()) {
				// File is same size and not newer in S3, skip
				needsDownload = false
				stats.Skipped++
			}
		}

		if !needsDownload {
			continue
		}

		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create directory for %s: %v\n", localPath, err)
			stats.Failed++
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
			stats.Failed++
			continue
		}

		// Create local file
		file, err := os.Create(localPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create file %s: %v\n", localPath, err)
			result.Body.Close()
			stats.Failed++
			continue
		}

		// Copy S3 object to local file
		_, err = io.Copy(file, result.Body)
		result.Body.Close()
		file.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write file %s: %v\n", localPath, err)
			stats.Failed++
			continue
		}

		// Update modification time to match S3
		if obj.LastModified != nil {
			os.Chtimes(localPath, *obj.LastModified, *obj.LastModified)
		}

		stats.Downloaded++
	}

	// Delete local files not in S3 if --delete flag is set
	if deleteLocal {
		fmt.Println("\nChecking for files to delete...")
		err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Get relative path from local directory
			relPath, err := filepath.Rel(localDir, path)
			if err != nil {
				return err
			}

			// Check if this file exists in S3
			if !s3Files[relPath] {
				fmt.Printf("Deleting %s (not in S3)\n", path)
				if err := os.Remove(path); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", path, err)
					return nil
				}
				stats.Deleted++
			}

			return nil
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during deletion walk: %v\n", err)
		}
	}

	return stats, nil
}
