package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Ollama   OllamaConfig   `mapstructure:"ollama"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Type     string `mapstructure:"type"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// OllamaConfig holds Ollama AI settings.
type OllamaConfig struct {
	URL     string        `mapstructure:"url"`
	Model   string        `mapstructure:"model"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Database: DatabaseConfig{
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			DBName:   "postgres",
			SSLMode:  "disable",
		},
		Ollama: OllamaConfig{
			URL:     "http://localhost:11434",
			Model:   "llama3.2",
			Timeout: 30 * time.Second,
		},
	}
}

// configDir returns the configuration directory path.
func configDir() string {
	if dir := os.Getenv("DBGENIUS_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".config", "dbgenius")
	}
	return filepath.Join(home, ".config", "dbgenius")
}

// configFile returns the path to the config file.
func configFile() string {
	return filepath.Join(configDir(), "config.toml")
}

// Load loads configuration from file and environment variables.
// Returns the merged Config, or an error if loading/validation fails.
func Load() (Config, error) {
	cfg := DefaultConfig()

	v := viper.New()
	v.SetConfigFile(configFile())
	v.SetConfigType("toml")

	// Environment variable overrides with DBGENIUS_ prefix
	v.SetEnvPrefix("DBGENIUS")
	v.AutomaticEnv()

	// Set defaults matching DefaultConfig
	v.SetDefault("database.type", cfg.Database.Type)
	v.SetDefault("database.host", cfg.Database.Host)
	v.SetDefault("database.port", cfg.Database.Port)
	v.SetDefault("database.user", cfg.Database.User)
	v.SetDefault("database.password", cfg.Database.Password)
	v.SetDefault("database.dbname", cfg.Database.DBName)
	v.SetDefault("database.sslmode", cfg.Database.SSLMode)
	v.SetDefault("ollama.url", cfg.Ollama.URL)
	v.SetDefault("ollama.model", cfg.Ollama.Model)
	v.SetDefault("ollama.timeout", cfg.Ollama.Timeout)

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found — use defaults + env vars
			// Ensure config directory exists for future writes
			_ = os.MkdirAll(configDir(), 0755)
		} else {
			return cfg, fmt.Errorf("error reading config: %w", err)
		}
	}

	// Unmarshal into config struct
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("error parsing config: %w", err)
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// Validate checks the configuration for valid values.
func (c Config) Validate() error {
	switch c.Database.Type {
	case "postgres", "sqlite", "mysql":
		// valid
	default:
		return fmt.Errorf("unsupported database type %q; expected postgres, sqlite, or mysql", c.Database.Type)
	}

	if c.Database.Host == "" && c.Database.Type != "sqlite" {
		return fmt.Errorf("database host is required for %s", c.Database.Type)
	}

	if c.Database.Port <= 0 {
		// Set default ports based on type
		switch c.Database.Type {
		case "postgres":
			c.Database.Port = 5432
		case "mysql":
			c.Database.Port = 3306
		case "sqlite":
			c.Database.Port = 0 // Not used for SQLite
		}
	}

	if c.Ollama.URL == "" {
		c.Ollama.URL = "http://localhost:11434"
	}

	if c.Ollama.Model == "" {
		c.Ollama.Model = "llama3.2"
	}

	if c.Ollama.Timeout <= 0 {
		c.Ollama.Timeout = 30 * time.Second
	}

	return nil
}

// SampleConfig returns the content of a sample config file.
func SampleConfig() string {
	return `# dbgenius configuration file

[database]
# Supported types: postgres, sqlite, mysql
type = "postgres"
host = "localhost"
port = 5432
user = "postgres"
password = ""
dbname = "postgres"
sslmode = "disable"

[ollama]
url = "http://localhost:11434"
model = "llama3.2"
timeout = "30s"
`
}

// WriteDefaultConfig creates a default config file if one doesn't exist.
func WriteDefaultConfig() error {
	path := configFile()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("cannot create config directory %s: %w", dir, err)
	}

	// Don't overwrite existing config
	if _, err := os.Stat(path); err == nil {
		return nil // File exists, don't overwrite
	}

	if err := os.WriteFile(path, []byte(SampleConfig()), 0644); err != nil {
		return fmt.Errorf("cannot write default config to %s: %w", path, err)
	}

	return nil
}
