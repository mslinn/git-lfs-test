package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the LFS test configuration
type Config struct {
	DatabasePath string `yaml:"database"`
	RemoteHost   string `yaml:"remote_host"`
	AutoRemote   bool   `yaml:"auto_remote"`
	TestDataPath string `yaml:"test_data"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, err := os.UserHomeDir()
	dbPath := "/home/mslinn/lfs_eval/lfs-test.db"
	if err == nil {
		dbPath = filepath.Join(homeDir, "lfs_eval", "lfs-test.db")
	}
	return &Config{
		DatabasePath: dbPath,
		RemoteHost:   "gojira",
		AutoRemote:   true,
		TestDataPath: "/mnt/f/work/git/git_lfs_test_data",
	}
}

// Load loads configuration from file and environment variables
// Priority: environment variables > config file > defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	configPath := os.Getenv("LFS_TEST_CONFIG")
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(homeDir, ".lfs-test-config")
		}
	}

	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			// Config file is optional, so we just skip if not found
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load config file: %w", err)
			}
		}
	}

	// Override with environment variables
	if db := os.Getenv("LFS_TEST_DB"); db != "" {
		cfg.DatabasePath = db
	}
	if host := os.Getenv("LFS_REMOTE_HOST"); host != "" {
		cfg.RemoteHost = host
	}
	if autoRemote := os.Getenv("LFS_AUTO_REMOTE"); autoRemote != "" {
		cfg.AutoRemote = autoRemote == "true" || autoRemote == "1"
	}
	if testData := os.Getenv("LFS_TEST_DATA"); testData != "" {
		cfg.TestDataPath = testData
	}

	return cfg, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// Save saves the configuration to a file
func (cfg *Config) Save(path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() string {
	configPath := os.Getenv("LFS_TEST_CONFIG")
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(homeDir, ".lfs-test-config")
		} else {
			configPath = ".lfs-test-config"
		}
	}
	return configPath
}

// IsRemoteHost returns true if the current hostname is not the remote host
func (cfg *Config) IsRemoteHost() bool {
	if !cfg.AutoRemote {
		return false
	}

	hostname, err := os.Hostname()
	if err != nil {
		return false
	}

	return hostname != cfg.RemoteHost
}

// GetDatabasePath returns the database path, expanding ~/ if needed
func (cfg *Config) GetDatabasePath() string {
	if len(cfg.DatabasePath) > 0 && cfg.DatabasePath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, cfg.DatabasePath[2:])
		}
	}
	return cfg.DatabasePath
}

// GetTestDataPath returns the test data path, expanding ~/ and environment variables
// Supports patterns like ~/path, $work/path, and ${work}/path
func (cfg *Config) GetTestDataPath() string {
	path := cfg.TestDataPath

	// Expand tilde
	if len(path) > 0 && path[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	// Expand environment variables ($VAR and ${VAR})
	path = os.ExpandEnv(path)

	return path
}

// ValidateRemoteHost checks if the remote host is accessible via SSH
// Returns nil if accessible or auto_remote is disabled
// Returns error with specific failure mode if inaccessible
func (cfg *Config) ValidateRemoteHost() error {
	// Skip validation if auto_remote is disabled
	if !cfg.AutoRemote {
		return nil
	}

	// Skip validation if remote host is empty
	if cfg.RemoteHost == "" {
		return fmt.Errorf("remote_host is empty but auto_remote is enabled")
	}

	// Skip validation if we're running on the remote host
	if !cfg.IsRemoteHost() {
		return nil // We're on the remote host, no need to check SSH
	}

	// Try to connect via SSH with a short timeout
	// Use BatchMode to avoid prompting for password
	// Use ConnectTimeout to fail quickly
	cmd := exec.Command("ssh",
		"-o", "ConnectTimeout=5",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		cfg.RemoteHost,
		"echo", "ok")

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Determine the specific failure mode
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode == 255 {
				// SSH connection failure (common exit code)
				return fmt.Errorf("cannot connect to remote_host '%s' via SSH: connection failed\n"+
					"Please verify:\n"+
					"  - Host is reachable on the network\n"+
					"  - SSH is running on the remote host\n"+
					"  - Firewall allows SSH connections\n"+
					"  - DNS resolves the hostname\n"+
					"Error: %v", cfg.RemoteHost, err)
			}
		}
		return fmt.Errorf("SSH connection to remote_host '%s' failed: %v\nOutput: %s",
			cfg.RemoteHost, err, string(output))
	}

	return nil
}

// ValidateDatabase checks if the database path is accessible
// Creates the directory if it doesn't exist
// Returns error if the path is invalid or inaccessible
func (cfg *Config) ValidateDatabase() error {
	dbPath := cfg.GetDatabasePath()

	if dbPath == "" {
		return fmt.Errorf("database path is empty")
	}

	// Get the directory containing the database
	dbDir := filepath.Dir(dbPath)

	// Try to create the directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("cannot create database directory '%s': %w", dbDir, err)
	}

	// Check if directory is writable by trying to create a temp file
	tempFile := filepath.Join(dbDir, ".write_test")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("database directory '%s' is not writable: %w", dbDir, err)
	}
	os.Remove(tempFile)

	return nil
}

// Validate performs comprehensive validation of all configuration parameters
// Returns error with details about any validation failures
func (cfg *Config) Validate() error {
	var errors []string

	// Validate database
	if err := cfg.ValidateDatabase(); err != nil {
		errors = append(errors, fmt.Sprintf("Database: %v", err))
	}

	// Validate remote host
	if err := cfg.ValidateRemoteHost(); err != nil {
		errors = append(errors, fmt.Sprintf("Remote host: %v", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  %s",
			filepath.Join(errors...))
	}

	return nil
}
