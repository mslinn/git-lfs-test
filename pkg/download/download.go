package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// FileDownload describes a file to download
type FileDownload struct {
	URL       string // Full URL to download from
	FileName  string // Target filename to save as
	URLDir    string // URL directory (for display purposes)
	ShortName string // Short name for display
}

// DownloadFile downloads a file from a URL with retry logic
// Returns true if the file was already present, false if it was downloaded
func DownloadFile(url, destPath string, debug bool) (bool, error) {
	// Check if file already exists
	if _, err := os.Stat(destPath); err == nil {
		if debug {
			fmt.Printf("  %s already exists\n", filepath.Base(destPath))
		}
		return true, nil
	}

	if debug {
		fmt.Printf("  Downloading %s\n", filepath.Base(destPath))
	}

	// Create parent directory if needed
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temporary file
	tempPath := destPath + ".download"
	out, err := os.Create(tempPath)
	if err != nil {
		return false, fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Download with retry logic
	const maxRetries = 5
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			if debug {
				fmt.Printf("  Retry %d/%d for %s\n", attempt-1, maxRetries-1, filepath.Base(destPath))
			}
			time.Sleep(time.Second * time.Duration(attempt))
		}

		// Make HTTP request
		client := &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large files
		}
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			continue
		}

		// Download the file
		_, err = io.Copy(out, resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			continue
		}

		// Success - rename temp file to final name
		out.Close()
		if err := os.Rename(tempPath, destPath); err != nil {
			return false, fmt.Errorf("failed to rename downloaded file: %w", err)
		}

		if debug {
			info, _ := os.Stat(destPath)
			fmt.Printf("  ✓ Downloaded %s (%s)\n", filepath.Base(destPath), formatSize(info.Size()))
		}

		return false, nil
	}

	// Clean up temp file on failure
	os.Remove(tempPath)

	return false, fmt.Errorf("failed after %d retries: %v", maxRetries, lastErr)
}

// formatSize formats a size in bytes as a human-readable string
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.0f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.0f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
