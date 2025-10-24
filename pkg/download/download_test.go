package download

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFile_AlreadyExists(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "existing.txt")

	// Create the file
	if err := os.WriteFile(destPath, []byte("already here"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to download - should return true (already exists)
	alreadyExists, err := DownloadFile("http://example.com/file.txt", destPath, false)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !alreadyExists {
		t.Errorf("Expected alreadyExists=true, got false")
	}

	// Verify file content hasn't changed
	content, _ := os.ReadFile(destPath)
	if string(content) != "already here" {
		t.Errorf("File content changed unexpectedly")
	}
}

func TestDownloadFile_Success(t *testing.T) {
	// Create test HTTP server
	expectedContent := "downloaded content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	// Create temporary directory
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "downloaded.txt")

	// Download file
	alreadyExists, err := DownloadFile(server.URL, destPath, false)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if alreadyExists {
		t.Errorf("Expected alreadyExists=false, got true")
	}

	// Verify file was created with correct content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content) != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, string(content))
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	// Create test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create temporary directory
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "notfound.txt")

	// Try to download - should fail
	_, err := DownloadFile(server.URL, destPath, false)
	if err == nil {
		t.Errorf("Expected error for 404 response, got nil")
	}

	// Verify file was not created
	if _, err := os.Stat(destPath); err == nil {
		t.Errorf("File should not exist after failed download")
	}
}

func TestDownloadFile_CreatesDirectory(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test"))
	}))
	defer server.Close()

	// Create temporary directory
	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "subdir", "nested", "file.txt")

	// Download file - should create parent directories
	_, err := DownloadFile(server.URL, destPath, false)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("File should exist at %s: %v", destPath, err)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1 KB"},
		{1536, "2 KB"},
		{1048576, "1 MB"},
		{1572864, "2 MB"},
		{1073741824, "1.00 GB"},
		{2147483648, "2.00 GB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}
