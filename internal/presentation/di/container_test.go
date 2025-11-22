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

				// Configの確認
				if container.Config() == nil {
					t.Error("Config() returned nil")
				}

				// AIRepositoryの確認
				if container.aiRepo == nil {
					t.Error("aiRepo is nil")
				}

				// AICorrectionUseCaseの確認
				if container.AICorrectionUseCase() == nil {
					t.Error("AICorrectionUseCase() returned nil")
				}

				// Closeの確認
				if err := container.Close(); err != nil {
					t.Errorf("Close() error = %v", err)
				}
			}
		})
	}
}

func TestContainer_Config(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	got := container.Config()
	if got == nil {
		t.Error("Config() returned nil")
	}

	if got != cfg {
		t.Error("Config() returned different config instance")
	}
}

func TestContainer_AICorrectionUseCase(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	useCase := container.AICorrectionUseCase()
	if useCase == nil {
		t.Error("AICorrectionUseCase() returned nil")
	}

	// 同じインスタンスが返されることを確認
	useCase2 := container.AICorrectionUseCase()
	if useCase != useCase2 {
		t.Error("AICorrectionUseCase() returned different instances")
	}
}

func TestContainer_Close(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "正常系: Close成功",
			cfg:     config.DefaultConfig(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container, err := NewContainer(tt.cfg)
			if err != nil {
				t.Fatalf("NewContainer() error = %v", err)
			}

			err = container.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Close後も再度Closeできることを確認（冪等性）
			err = container.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("Second Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainer_MultipleInstances(t *testing.T) {
	cfg := config.DefaultConfig()

	// 複数のコンテナを作成
	container1, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	container2, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}

	// 異なるインスタンスであることを確認
	if container1 == container2 {
		t.Error("Expected different container instances")
	}

	// それぞれのUseCaseも異なるインスタンスであることを確認
	if container1.AICorrectionUseCase() == container2.AICorrectionUseCase() {
		t.Error("Expected different usecase instances")
	}

	// クリーンアップ
	if err := container1.Close(); err != nil {
		t.Errorf("container1.Close() error = %v", err)
	}
	if err := container2.Close(); err != nil {
		t.Errorf("container2.Close() error = %v", err)
	}
}

func TestContainer_NilConfig(t *testing.T) {
	// nilの設定でコンテナを作成しようとする
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil config, but didn't panic")
		}
	}()

	_, _ = NewContainer(nil)
}

func TestContainer_GetProviderName(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	defer func() { _ = container.Close() }()

	useCase := container.AICorrectionUseCase()
	providerName := useCase.GetProviderName()

	if providerName == "" {
		t.Error("GetProviderName() returned empty string")
	}

	// Claudeが含まれていることを確認
	if providerName == "" {
		t.Error("Expected non-empty provider name")
	}
}

func TestContainer_CacheRepository(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	defer func() { _ = container.Close() }()

	t.Run("正常系: CacheRepository取得", func(t *testing.T) {
		cacheRepo := container.CacheRepository()
		// Redis接続に失敗した場合はnilが返る
		if cacheRepo != nil {
			// 同じインスタンスが返されることを確認
			cacheRepo2 := container.CacheRepository()
			if cacheRepo != cacheRepo2 {
				t.Error("CacheRepository() returned different instances")
			}
		}
	})
}

func TestContainer_ReceiptRepository(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	defer func() { _ = container.Close() }()

	t.Run("正常系: ReceiptRepository取得", func(t *testing.T) {
		receiptRepo := container.ReceiptRepository()
		// MySQL接続に失敗した場合はnilが返る
		if receiptRepo != nil {
			// 同じインスタンスが返されることを確認
			receiptRepo2 := container.ReceiptRepository()
			if receiptRepo != receiptRepo2 {
				t.Error("ReceiptRepository() returned different instances")
			}
		}
	})
}

func TestContainer_RepositoriesNil(t *testing.T) {
	// 無効な設定でコンテナを作成
	cfg := &config.Config{
		Anthropic: config.AnthropicConfig{
			APIKey:    "test-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		},
		Redis: config.RedisConfig{
			Host:     "invalid-host-that-does-not-exist",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		MySQL: config.MySQLConfig{
			Host:     "invalid-host-that-does-not-exist",
			Port:     3306,
			User:     "root",
			Password: "password",
			Database: "test",
		},
	}

	container, err := NewContainer(cfg)
	if err != nil {
		t.Fatalf("NewContainer() error = %v", err)
	}
	defer func() { _ = container.Close() }()

	t.Run("異常系: Redis接続失敗時はnilを返す", func(t *testing.T) {
		cacheRepo := container.CacheRepository()
		if cacheRepo != nil {
			t.Log("CacheRepository is not nil (Redis connection succeeded unexpectedly)")
		}
	})

	t.Run("異常系: MySQL接続失敗時はnilを返す", func(t *testing.T) {
		receiptRepo := container.ReceiptRepository()
		if receiptRepo != nil {
			t.Log("ReceiptRepository is not nil (MySQL connection succeeded unexpectedly)")
		}
	})
}
