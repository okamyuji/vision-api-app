package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config アプリケーション全体の設定
type Config struct {
	Anthropic AnthropicConfig `yaml:"anthropic"`
	Redis     RedisConfig     `yaml:"redis"`
	MySQL     MySQLConfig     `yaml:"mysql"`
}

// AnthropicConfig Anthropic APIの設定
type AnthropicConfig struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// RedisConfig Redisの設定
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// MySQLConfig MySQLの設定
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// Load 設定ファイルを読み込む
func Load(configPath string) (*Config, error) {
	// 設定ファイルが存在しない場合はデフォルト設定を返す
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 環境変数の展開
	dataStr := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(dataStr), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// DefaultConfig デフォルト設定を返す
func DefaultConfig() *Config {
	// Redis/MySQLのホストはテスト環境では localhost を使用
	redisHost := "redis"
	mysqlHost := "mysql"
	if os.Getenv("GO_ENV") == "test" {
		redisHost = "localhost"
		mysqlHost = "localhost"
	}

	return &Config{
		Anthropic: AnthropicConfig{
			APIKey:    os.Getenv("ANTHROPIC_API_KEY"),
			Model:     "claude-haiku-4-5-20251001",
			MaxTokens: 4096,
		},
		Redis: RedisConfig{
			Host:     redisHost,
			Port:     6379,
			Password: "",
			DB:       0,
		},
		MySQL: MySQLConfig{
			Host:     mysqlHost,
			Port:     3306,
			User:     "root",
			Password: os.Getenv("MYSQL_ROOT_PASSWORD"),
			Database: "household",
		},
	}
}

// Save 設定をファイルに保存する
func (c *Config) Save(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
