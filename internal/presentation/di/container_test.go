package di

import (
	"testing"

	"vision-api-app/internal/config"
)

func TestNewContainer(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "正常系: デフォルト設定",
			cfg:     config.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "正常系: カスタム設定",
			cfg: &config.Config{
				Anthropic: config.AnthropicConfig{
					APIKey:    "test-api-key",
					Model:     "claude-3-haiku-20240307",
					MaxTokens: 4096,
				},
				Redis: config.RedisConfig{
					Host:     "localhost",
					Port:     6379,
					Password: "",
					DB:       0,
				},
				MySQL: config.MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "root",
					Password: "password",
					Database: "test",
				},
			},
			wantErr: false,
		},
		{
			name: "正常系: 空のAPIキー（初期化は成功）",
			cfg: &config.Config{
				Anthropic: config.AnthropicConfig{
					APIKey:    "",
					Model:     "claude-3-haiku-20240307",
					MaxTokens: 4096,
				},
				Redis: config.RedisConfig{
					Host:     "localhost",
					Port:     6379,
					Password: "",
					DB:       0,
				},
				MySQL: config.MySQLConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "root",
					Password: "password",
					Database: "test",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, err := NewContainer(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if container == nil {
					t.Error("Expected non-nil container")
					return
				}

				// AIRepositoryの確認
				if container.aiRepo == nil {
					t.Error("aiRepo is nil")
				}

				// AICorrectionUseCaseの確認
				if container.AICorrectionUseCase() == nil {
					t.Error("AICorrectionUseCase() returned nil")
				}

				// VisionHandlerの確認
				if container.VisionHandler() == nil {
					t.Error("VisionHandler() returned nil")
				}

				// WebHandlerの確認
				if container.WebHandler() == nil {
					t.Error("WebHandler() returned nil")
				}

				// Closeの確認
				if err := container.Close(); err != nil {
					t.Errorf("Close() error = %v", err)
				}
			}
		})
	}
}

func TestContainer_Close(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	// Closeを実行
	if err := container.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 2回目のCloseも問題なく実行できることを確認
	if err := container.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}
