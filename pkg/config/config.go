package config

import (
	"fmt"
	"os"
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
