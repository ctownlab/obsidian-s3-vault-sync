package tar

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

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

// CreateVaultTarball creates a tarball of the vault with automatic filename generation
func CreateVaultTarball(localDir, vaultName, tarDir string) (string, error) {
	// Generate tarball filename from vault name and timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	tarFilename := fmt.Sprintf("%s-%s.tar.gz", vaultName, timestamp)

	// Determine output directory
	outputDir := "."
	if tarDir != "" {
		outputDir = tarDir
		// Create tar directory if it doesn't exist
		if err := os.MkdirAll(tarDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create tar directory: %w", err)
		}
	}

	tarOutput := filepath.Join(outputDir, tarFilename)

	slog.Debug("Creating tarball", "path", tarOutput)
	if err := createTarball(localDir, tarOutput); err != nil {
		return "", fmt.Errorf("failed to create tarball: %w", err)
	}

	// Get tarball size
	tarInfo, err := os.Stat(tarOutput)
	if err == nil {
		sizeMB := float64(tarInfo.Size()) / (1024 * 1024)
		slog.Info("Tarball created",
			"path", tarOutput,
			"size_mb", fmt.Sprintf("%.2f", sizeMB))
	} else {
		slog.Info("Tarball created", "path", tarOutput)
	}

	return outputDir, nil
}

// CleanupOldTarballs removes old tarballs keeping only the specified number
func CleanupOldTarballs(outputDir, vaultName string, keepCount int) error {
	if keepCount <= 0 {
		return nil
	}

	slog.Debug("Checking for old tarballs to clean up", "keep_count", keepCount)

	// Find all tarballs matching the pattern in the output directory
	pattern := filepath.Join(outputDir, fmt.Sprintf("%s-*.tar.gz", vaultName))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to find old tarballs: %w", err)
	}

	if len(matches) <= keepCount {
		// Nothing to clean up
		return nil
	}

	// Sort by modification time (oldest first)
	type tarballInfo struct {
		path    string
		modTime time.Time
	}
	var tarballs []tarballInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil {
			tarballs = append(tarballs, tarballInfo{path: match, modTime: info.ModTime()})
		}
	}

	// Sort by modification time (bubble sort for simplicity)
	for i := 0; i < len(tarballs)-1; i++ {
		for j := i + 1; j < len(tarballs); j++ {
			if tarballs[i].modTime.After(tarballs[j].modTime) {
				tarballs[i], tarballs[j] = tarballs[j], tarballs[i]
			}
		}
	}

	// Delete oldest tarballs beyond keep count
	deleteCount := len(tarballs) - keepCount
	for i := 0; i < deleteCount; i++ {
		slog.Debug("Deleting old tarball", "path", tarballs[i].path)
		if err := os.Remove(tarballs[i].path); err != nil {
			slog.Warn("Failed to delete tarball",
				"path", tarballs[i].path,
				"error", err)
		}
	}
	slog.Info("Cleaned up old tarballs", "count", deleteCount)

	return nil
}
