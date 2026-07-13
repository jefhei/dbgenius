package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Database.Type != "postgres" {
		t.Errorf("Default DB type = %q, want postgres", cfg.Database.Type)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("Default host = %q, want localhost", cfg.Database.Host)
	}
	if cfg.Database.Port != 5432 {
		t.Errorf("Default port = %d, want 5432", cfg.Database.Port)
	}
	if cfg.Database.User != "postgres" {
		t.Errorf("Default user = %q, want postgres", cfg.Database.User)
	}
	if cfg.Database.DBName != "postgres" {
		t.Errorf("Default dbname = %q, want postgres", cfg.Database.DBName)
	}
	if cfg.Database.SSLMode != "disable" {
		t.Errorf("Default sslmode = %q, want disable", cfg.Database.SSLMode)
	}
	if cfg.Ollama.URL != "http://localhost:11434" {
		t.Errorf("Default Ollama URL = %q, want http://localhost:11434", cfg.Ollama.URL)
	}
	if cfg.Ollama.Model != "llama3.2" {
		t.Errorf("Default Ollama model = %q, want llama3.2", cfg.Ollama.Model)
	}
	if cfg.Ollama.Timeout != 30*time.Second {
		t.Errorf("Default Ollama timeout = %v, want 30s", cfg.Ollama.Timeout)
	}
}

func TestValidate_ValidTypes(t *testing.T) {
	for _, dbType := range []string{"postgres", "sqlite", "mysql"} {
		cfg := DefaultConfig()
		cfg.Database.Type = dbType
		if dbType == "sqlite" {
			cfg.Database.Host = "" // SQLite doesn't need host
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate(%q) failed: %v", dbType, err)
		}
	}
}

func TestValidate_InvalidType(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Type = "invalid"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for invalid DB type")
	}
	if !strings.Contains(err.Error(), "unsupported database type") {
		t.Errorf("Error = %q, want 'unsupported database type'", err.Error())
	}
}

func TestValidate_EmptyHost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Host = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected error for empty host with postgres")
	}
	if !strings.Contains(err.Error(), "host is required") {
		t.Errorf("Error = %q, want 'host is required'", err.Error())
	}
}

func TestValidate_SQLiteNoHost(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Type = "sqlite"
	cfg.Database.Host = ""
	err := cfg.Validate()
	if err != nil {
		t.Errorf("SQLite without host should be valid: %v", err)
	}
}

func TestValidate_EmptyOllamaURL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Ollama.URL = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("Empty Ollama URL should be valid (defaults applied): %v", err)
	}
}

func TestValidate_EmptyOllamaModel(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Ollama.Model = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("Empty Ollama model should be valid (defaults applied): %v", err)
	}
}

func TestValidate_ZeroTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Ollama.Timeout = 0
	if err := cfg.Validate(); err != nil {
		t.Errorf("Zero timeout should be valid (defaults applied): %v", err)
	}
}

func TestValidate_NegativePort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Port = -1
	if err := cfg.Validate(); err != nil {
		t.Errorf("Port -1 should be valid (defaults applied): %v", err)
	}
}

func TestValidate_MySQLPortDefault(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Database.Type = "mysql"
	cfg.Database.Port = -1
	if err := cfg.Validate(); err != nil {
		t.Errorf("MySQL with port -1 should be valid: %v", err)
	}
}

func TestConfigDir_EnvVar(t *testing.T) {
	os.Setenv("DBGENIUS_CONFIG_DIR", "/tmp/dbgenius-test-config")
	defer os.Unsetenv("DBGENIUS_CONFIG_DIR")

	dir := configDir()
	if dir != "/tmp/dbgenius-test-config" {
		t.Errorf("configDir = %q, want /tmp/dbgenius-test-config", dir)
	}
}

func TestSampleConfig(t *testing.T) {
	sample := SampleConfig()
	if !strings.Contains(sample, "[database]") {
		t.Error("Sample config should contain [database] section")
	}
	if !strings.Contains(sample, "[ollama]") {
		t.Error("Sample config should contain [ollama] section")
	}
	if !strings.Contains(sample, "type = \"postgres\"") {
		t.Error("Sample config should contain type = \"postgres\"")
	}
}

func TestWriteDefaultConfig(t *testing.T) {
	// Use a temp dir
	tmpDir := t.TempDir()
	os.Setenv("DBGENIUS_CONFIG_DIR", tmpDir)
	defer os.Unsetenv("DBGENIUS_CONFIG_DIR")

	if err := WriteDefaultConfig(); err != nil {
		t.Fatalf("WriteDefaultConfig failed: %v", err)
	}

	configPath := filepath.Join(tmpDir, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Default config file was not created")
	}

	// Write again — should not overwrite existing
	if err := WriteDefaultConfig(); err != nil {
		t.Fatalf("WriteDefaultConfig (exists) failed: %v", err)
	}
}

func TestWriteDefaultConfig_Subdir(t *testing.T) {
	// Use a temp dir with a nested subdirectory to test MkdirAll
	tmpDir := t.TempDir()
	os.Setenv("DBGENIUS_CONFIG_DIR", filepath.Join(tmpDir, "nested", "dir"))
	defer os.Unsetenv("DBGENIUS_CONFIG_DIR")

	if err := WriteDefaultConfig(); err != nil {
		t.Fatalf("WriteDefaultConfig should create nested dirs: %v", err)
	}

	configPath := filepath.Join(tmpDir, "nested", "dir", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Default config file was not created in nested dir")
	}
}
