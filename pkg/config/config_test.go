package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.DatabasePath == "" {
		t.Error("DatabasePath should not be empty")
	}

	if cfg.RemoteHost != "gojira" {
		t.Errorf("Expected RemoteHost='gojira', got '%s'", cfg.RemoteHost)
	}

	if !cfg.AutoRemote {
		t.Error("AutoRemote should be true by default")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create and save a config
	cfg := &Config{
		DatabasePath: "/tmp/test.db",
		RemoteHost:   "testhost",
		AutoRemote:   false,
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load the config back
	loadedCfg := DefaultConfig()
	if err := loadFromFile(loadedCfg, configPath); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if loadedCfg.DatabasePath != cfg.DatabasePath {
		t.Errorf("DatabasePath mismatch: expected '%s', got '%s'", cfg.DatabasePath, loadedCfg.DatabasePath)
	}
	if loadedCfg.RemoteHost != cfg.RemoteHost {
		t.Errorf("RemoteHost mismatch: expected '%s', got '%s'", cfg.RemoteHost, loadedCfg.RemoteHost)
	}
	if loadedCfg.AutoRemote != cfg.AutoRemote {
		t.Errorf("AutoRemote mismatch: expected %v, got %v", cfg.AutoRemote, loadedCfg.AutoRemote)
	}
}

func TestLoadWithEnvironmentOverrides(t *testing.T) {
	// Save original environment
	origDB := os.Getenv("LFS_TEST_DB")
	origHost := os.Getenv("LFS_REMOTE_HOST")
	origAuto := os.Getenv("LFS_AUTO_REMOTE")
	origConfig := os.Getenv("LFS_TEST_CONFIG")
	defer func() {
		os.Setenv("LFS_TEST_DB", origDB)
		os.Setenv("LFS_REMOTE_HOST", origHost)
		os.Setenv("LFS_AUTO_REMOTE", origAuto)
		os.Setenv("LFS_TEST_CONFIG", origConfig)
	}()

	// Set environment variables
	os.Setenv("LFS_TEST_DB", "/env/test.db")
	os.Setenv("LFS_REMOTE_HOST", "envhost")
	os.Setenv("LFS_AUTO_REMOTE", "false")
	os.Setenv("LFS_TEST_CONFIG", "/nonexistent/config")

	// Load config (will use defaults + env overrides)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment overrides
	if cfg.DatabasePath != "/env/test.db" {
		t.Errorf("Expected DatabasePath from env '/env/test.db', got '%s'", cfg.DatabasePath)
	}
	if cfg.RemoteHost != "envhost" {
		t.Errorf("Expected RemoteHost from env 'envhost', got '%s'", cfg.RemoteHost)
	}
	if cfg.AutoRemote {
		t.Error("Expected AutoRemote to be false from env")
	}
}

func TestGetDatabasePath(t *testing.T) {
	tests := []struct {
		name     string
		dbPath   string
		expected func() string
	}{
		{
			name:   "absolute path",
			dbPath: "/absolute/path/to/db",
			expected: func() string {
				return "/absolute/path/to/db"
			},
		},
		{
			name:   "home directory expansion",
			dbPath: "~/lfs_eval/test.db",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "lfs_eval/test.db")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{DatabasePath: tt.dbPath}
			got := cfg.GetDatabasePath()
			expected := tt.expected()
			if got != expected {
				t.Errorf("GetDatabasePath() = %v, want %v", got, expected)
			}
		})
	}
}

func TestIsRemoteHost(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Skip("Cannot get hostname, skipping test")
	}

	tests := []struct {
		name       string
		remoteHost string
		autoRemote bool
		expected   bool
	}{
		{
			name:       "same host with auto_remote",
			remoteHost: hostname,
			autoRemote: true,
			expected:   false, // Not remote (same host)
		},
		{
			name:       "different host with auto_remote",
			remoteHost: "different-host",
			autoRemote: true,
			expected:   true, // Is remote (different host)
		},
		{
			name:       "auto_remote disabled",
			remoteHost: "different-host",
			autoRemote: false,
			expected:   false, // Not remote (auto_remote disabled)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				RemoteHost: tt.remoteHost,
				AutoRemote: tt.autoRemote,
			}
			got := cfg.IsRemoteHost()
			if got != tt.expected {
				t.Errorf("IsRemoteHost() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Save original
	orig := os.Getenv("LFS_TEST_CONFIG")
	defer os.Setenv("LFS_TEST_CONFIG", orig)

	// Test with environment variable
	os.Setenv("LFS_TEST_CONFIG", "/custom/config/path")
	path := GetConfigPath()
	if path != "/custom/config/path" {
		t.Errorf("GetConfigPath() with env = %v, want /custom/config/path", path)
	}

	// Test without environment variable (should use home dir)
	os.Setenv("LFS_TEST_CONFIG", "")
	path = GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath() should not return empty string")
	}
	if !filepath.IsAbs(path) && path != ".lfs-test-config" {
		t.Errorf("GetConfigPath() should return absolute path or relative fallback, got %v", path)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Try to save to a nested path that doesn't exist
	configPath := filepath.Join(tempDir, "nested", "dir", "config.yaml")

	cfg := DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save should create parent directories: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created in nested directory")
	}
}

func TestGetTestDataPath(t *testing.T) {
	tests := []struct {
		name     string
		testData string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "absolute path",
			testData: "/absolute/path/to/data",
			expected: "/absolute/path/to/data",
		},
		{
			name:     "home directory expansion",
			testData: "~/test_data",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "test_data")
			}(),
		},
		{
			name:     "environment variable expansion",
			testData: "$work/git/git_lfs_test_data",
			envVars: map[string]string{
				"work": "/mnt/f/work",
			},
			expected: "/mnt/f/work/git/git_lfs_test_data",
		},
		{
			name:     "environment variable with braces",
			testData: "${work}/git/git_lfs_test_data",
			envVars: map[string]string{
				"work": "/mnt/f/work",
			},
			expected: "/mnt/f/work/git/git_lfs_test_data",
		},
		{
			name:     "multiple environment variables",
			testData: "$base/$subdir/data",
			envVars: map[string]string{
				"base":   "/mnt/f/work",
				"subdir": "git",
			},
			expected: "/mnt/f/work/git/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and set environment variables
			savedEnvVars := make(map[string]string)
			for key, value := range tt.envVars {
				savedEnvVars[key] = os.Getenv(key)
				os.Setenv(key, value)
			}
			defer func() {
				for key, value := range savedEnvVars {
					os.Setenv(key, value)
				}
			}()

			cfg := &Config{TestDataPath: tt.testData}
			got := cfg.GetTestDataPath()
			if got != tt.expected {
				t.Errorf("GetTestDataPath() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetTestDataPath_UndefinedVariable(t *testing.T) {
	// Ensure the variable doesn't exist
	os.Unsetenv("UNDEFINED_VAR_FOR_TEST")

	cfg := &Config{TestDataPath: "$UNDEFINED_VAR_FOR_TEST/data"}
	got := cfg.GetTestDataPath()

	// os.ExpandEnv expands undefined variables to empty string
	// So $UNDEFINED_VAR_FOR_TEST/data becomes /data
	expected := "/data"
	if got != expected {
		t.Errorf("GetTestDataPath() with undefined var = %v, want %v", got, expected)
	}
}

func TestValidateDatabase(t *testing.T) {
	tests := []struct {
		name      string
		dbPath    string
		wantError bool
	}{
		{
			name:      "valid path",
			dbPath:    filepath.Join(os.TempDir(), "test_db", "test.db"),
			wantError: false,
		},
		{
			name:      "empty path",
			dbPath:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{DatabasePath: tt.dbPath}
			err := cfg.ValidateDatabase()

			if tt.wantError && err == nil {
				t.Error("ValidateDatabase() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("ValidateDatabase() unexpected error: %v", err)
			}

			// Clean up test directory if created
			if !tt.wantError && tt.dbPath != "" {
				os.RemoveAll(filepath.Dir(tt.dbPath))
			}
		})
	}
}

func TestValidateDatabase_CreatesDirectory(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "db_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Use a nested path that doesn't exist yet
	dbPath := filepath.Join(tempDir, "nested", "dir", "test.db")
	cfg := &Config{DatabasePath: dbPath}

	// Validation should create the directory
	if err := cfg.ValidateDatabase(); err != nil {
		t.Fatalf("ValidateDatabase() failed: %v", err)
	}

	// Verify directory was created
	dbDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		t.Error("ValidateDatabase() did not create database directory")
	}
}

func TestValidateRemoteHost_AutoRemoteDisabled(t *testing.T) {
	cfg := &Config{
		RemoteHost: "nonexistent-host",
		AutoRemote: false,
	}

	// Should not error when auto_remote is disabled
	if err := cfg.ValidateRemoteHost(); err != nil {
		t.Errorf("ValidateRemoteHost() with auto_remote=false should not error, got: %v", err)
	}
}

func TestValidateRemoteHost_EmptyHost(t *testing.T) {
	cfg := &Config{
		RemoteHost: "",
		AutoRemote: true,
	}

	// Should error when remote_host is empty but auto_remote is enabled
	err := cfg.ValidateRemoteHost()
	if err == nil {
		t.Error("ValidateRemoteHost() with empty remote_host should return error")
	}
	if err != nil && !contains(err.Error(), "empty") {
		t.Errorf("Error should mention 'empty', got: %v", err)
	}
}

func TestValidateRemoteHost_SameHost(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Skip("Cannot get hostname, skipping test")
	}

	cfg := &Config{
		RemoteHost: hostname,
		AutoRemote: true,
	}

	// Should not error when running on the remote host itself
	if err := cfg.ValidateRemoteHost(); err != nil {
		t.Errorf("ValidateRemoteHost() on same host should not error, got: %v", err)
	}
}

func TestValidate_Comprehensive(t *testing.T) {
	// Create a valid temporary database path
	tempDir, err := os.MkdirTemp("", "validate_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	hostname, err := os.Hostname()
	if err != nil {
		t.Skip("Cannot get hostname, skipping test")
	}

	tests := []struct {
		name      string
		cfg       *Config
		wantError bool
	}{
		{
			name: "valid configuration",
			cfg: &Config{
				DatabasePath: filepath.Join(tempDir, "test.db"),
				RemoteHost:   hostname, // Same as current host
				AutoRemote:   true,
			},
			wantError: false,
		},
		{
			name: "invalid database path",
			cfg: &Config{
				DatabasePath: "",
				RemoteHost:   hostname,
				AutoRemote:   true,
			},
			wantError: true,
		},
		{
			name: "auto_remote disabled",
			cfg: &Config{
				DatabasePath: filepath.Join(tempDir, "test.db"),
				RemoteHost:   "any-host",
				AutoRemote:   false,
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantError && err == nil {
				t.Error("Validate() expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
