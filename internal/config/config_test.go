package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if cfg.Anthropic.Model == "" {
		t.Error("Expected non-empty model")
	}

	if cfg.Redis.Port <= 0 {
		t.Error("Expected positive Redis port")
	}

	if cfg.MySQL.Port <= 0 {
		t.Error("Expected positive MySQL port")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected default config, got nil")
	}
}

func TestSave(t *testing.T) {
	cfg := DefaultConfig()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// ファイルが存在することを確認
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// 読み込んで確認
	loadedCfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loadedCfg.Anthropic.Model != cfg.Anthropic.Model {
		t.Error("Loaded config does not match saved config")
	}
}

func TestSave_InvalidPath(t *testing.T) {
	cfg := DefaultConfig()
	// 無効なパス（書き込み不可）
	err := cfg.Save("/invalid/path/that/does/not/exist/config.yaml")
	if err == nil {
		t.Error("Expected error for invalid path, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// 無効なYAMLファイルを作成
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid YAML file: %v", err)
	}

	// 無効なYAMLの場合はエラーを返すことを確認
	_, err = Load(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
