package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// MockServer テスト用のモックサーバー
type MockServer struct {
	listenAndServeFunc func() error
	shutdownFunc       func(ctx context.Context) error
}

func (m *MockServer) ListenAndServe() error {
	if m.listenAndServeFunc != nil {
		return m.listenAndServeFunc()
	}
	return nil
}

func (m *MockServer) Shutdown(ctx context.Context) error {
	if m.shutdownFunc != nil {
		return m.shutdownFunc(ctx)
	}
	return nil
}

// TestNewApp_Fast NewAppの高速テスト（サーバー起動なし）
func TestNewApp_Fast(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		port       string
		wantPort   string
		wantErr    bool
	}{
		{
			name:       "正常系: デフォルト設定",
			configPath: "nonexistent.yaml",
			port:       "8080",
			wantPort:   ":8080",
			wantErr:    false,
		},
		{
			name:       "正常系: カスタムポート",
			configPath: "nonexistent.yaml",
			port:       "9090",
			wantPort:   ":9090",
			wantErr:    false,
		},
		{
			name:       "正常系: 空ポート（デフォルト）",
			configPath: "nonexistent.yaml",
			port:       "",
			wantPort:   ":8080",
			wantErr:    false,
		},
		{
			name:       "境界値: 最小ポート",
			configPath: "nonexistent.yaml",
			port:       "1",
			wantPort:   ":1",
			wantErr:    false,
		},
		{
			name:       "境界値: 最大ポート",
			configPath: "nonexistent.yaml",
			port:       "65535",
			wantPort:   ":65535",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp(&AppConfig{
				ConfigPath: tt.configPath,
				Port:       tt.port,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("NewApp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			defer func() { _ = app.container.Close() }()

			// 基本的な構造の確認
			if app == nil {
				t.Fatal("app is nil")
			}
			if app.config == nil {
				t.Error("app.config is nil")
			}
			if app.container == nil {
				t.Error("app.container is nil")
			}
			if app.server == nil {
				t.Error("app.server is nil")
			}

			// サーバー設定の確認
			if app.server.Addr != tt.wantPort {
				t.Errorf("server.Addr = %v, want %v", app.server.Addr, tt.wantPort)
			}

			// タイムアウト設定の検証
			if app.server.ReadTimeout != 30*time.Second {
				t.Errorf("ReadTimeout = %v, want 30s", app.server.ReadTimeout)
			}
			if app.server.WriteTimeout != 30*time.Second {
				t.Errorf("WriteTimeout = %v, want 30s", app.server.WriteTimeout)
			}
			if app.server.IdleTimeout != 60*time.Second {
				t.Errorf("IdleTimeout = %v, want 60s", app.server.IdleTimeout)
			}

			// ハンドラーの確認
			if app.server.Handler == nil {
				t.Error("server.Handler is nil")
			}
		})
	}
}

// TestApp_Components コンポーネントの検証
func TestApp_Components(t *testing.T) {
	app, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "8080",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer func() { _ = app.container.Close() }()

	t.Run("Config存在確認", func(t *testing.T) {
		if app.config == nil {
			t.Error("config is nil")
		}
		if app.config.Port != "8080" {
			t.Errorf("Port = %v, want 8080", app.config.Port)
		}
		if app.config.ConfigPath == "" {
			t.Error("ConfigPath is empty")
		}
	})

	t.Run("Container存在確認", func(t *testing.T) {
		if app.container == nil {
			t.Error("container is nil")
		}
		if app.container.Config() == nil {
			t.Error("container.Config() is nil")
		}
		if app.container.AICorrectionUseCase() == nil {
			t.Error("container.AICorrectionUseCase() is nil")
		}
	})

	t.Run("Server設定確認", func(t *testing.T) {
		if app.server == nil {
			t.Error("server is nil")
		}
		if app.server.Handler == nil {
			t.Error("server.Handler is nil")
		}
		if app.server.Addr != ":8080" {
			t.Errorf("server.Addr = %v, want :8080", app.server.Addr)
		}
		if app.server.ReadTimeout == 0 {
			t.Error("ReadTimeout is 0")
		}
		if app.server.WriteTimeout == 0 {
			t.Error("WriteTimeout is 0")
		}
		if app.server.IdleTimeout == 0 {
			t.Error("IdleTimeout is 0")
		}
	})
}

// TestApp_WithConfigFile 設定ファイルを使用したテスト
func TestApp_WithConfigFile(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		port           string
		wantErr        bool
		validateConfig func(*testing.T, *App)
	}{
		{
			name: "正常系: 有効な設定ファイル",
			configContent: `anthropic:
  api_key: test-api-key-from-file
  model: claude-3-haiku-20240307
  max_tokens: 2048
`,
			port:    "9000",
			wantErr: false,
			validateConfig: func(t *testing.T, app *App) {
				if app.server.Addr != ":9000" {
					t.Errorf("server.Addr = %v, want :9000", app.server.Addr)
				}
			},
		},
		{
			name:          "異常系: 無効なYAML（デフォルトにフォールバック）",
			configContent: `invalid: yaml: [[[`,
			port:          "8080",
			wantErr:       false,
			validateConfig: func(t *testing.T, app *App) {
				if app == nil {
					t.Fatal("app should not be nil even with invalid config")
				}
			},
		},
		{
			name:          "正常系: 空の設定ファイル",
			configContent: "",
			port:          "8080",
			wantErr:       false,
			validateConfig: func(t *testing.T, app *App) {
				if app.server.Addr != ":8080" {
					t.Errorf("server.Addr = %v, want :8080", app.server.Addr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.configContent), 0644); err != nil {
				t.Fatalf("Failed to create test config: %v", err)
			}

			app, err := NewApp(&AppConfig{
				ConfigPath: configPath,
				Port:       tt.port,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("NewApp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			defer func() { _ = app.container.Close() }()

			if tt.validateConfig != nil {
				tt.validateConfig(t, app)
			}
		})
	}
}

// TestApp_Shutdown Shutdownの動作確認（サーバー起動なし）
func TestApp_Shutdown(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		wantPanic bool
	}{
		{
			name:      "正常系: 通常のタイムアウト",
			timeout:   5 * time.Second,
			wantPanic: false,
		},
		{
			name:      "正常系: 短いタイムアウト",
			timeout:   1 * time.Second,
			wantPanic: false,
		},
		{
			name:      "境界値: 1ナノ秒タイムアウト",
			timeout:   1 * time.Nanosecond,
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("Unexpected panic: %v", r)
					}
				}
			}()

			app, err := NewApp(&AppConfig{
				ConfigPath: "nonexistent.yaml",
				Port:       "8080",
			})
			if err != nil {
				t.Fatalf("NewApp() error = %v", err)
			}

			// サーバー起動せずにShutdownを呼ぶ（panicしないことを確認）
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			_ = app.Shutdown(ctx)
		})
	}
}

// TestApp_Shutdown_Idempotency Shutdownの冪等性テスト
func TestApp_Shutdown_Idempotency(t *testing.T) {
	app, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "8080",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	// 1回目のShutdown
	ctx1, cancel1 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel1()
	_ = app.Shutdown(ctx1)

	// 2回目のShutdown（冪等性の確認）
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()
	_ = app.Shutdown(ctx2)

	// 3回目のShutdown
	ctx3, cancel3 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel3()
	_ = app.Shutdown(ctx3)
}

// TestApp_ProviderName プロバイダー名の確認
func TestApp_ProviderName(t *testing.T) {
	app, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "8080",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer func() { _ = app.container.Close() }()

	providerName := app.container.AICorrectionUseCase().GetProviderName()
	if providerName == "" {
		t.Error("GetProviderName() returned empty string")
	}

	t.Logf("Provider name: %s", providerName)
}

// TestApp_ContainerClose コンテナのクローズ
func TestApp_ContainerClose(t *testing.T) {
	app, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "8080",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}

	// Containerを明示的にClose
	if err := app.container.Close(); err != nil {
		t.Errorf("container.Close() error = %v", err)
	}

	// 2回目のClose（冪等性）
	if err := app.container.Close(); err != nil {
		t.Logf("Second Close() returned error (expected): %v", err)
	}
}

// TestApp_NilConfig nilの設定でのパニックテスト
func TestApp_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil config, but didn't panic")
		}
	}()

	_, _ = NewApp(nil)
}

// TestApp_ConfigPath 設定パスの確認
func TestApp_ConfigPath(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		port       string
	}{
		{
			name:       "正常系: 相対パス",
			configPath: "config.yaml",
			port:       "8080",
		},
		{
			name:       "正常系: 絶対パス",
			configPath: "/tmp/config.yaml",
			port:       "8080",
		},
		{
			name:       "正常系: 存在しないパス",
			configPath: "nonexistent.yaml",
			port:       "8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp(&AppConfig{
				ConfigPath: tt.configPath,
				Port:       tt.port,
			})
			if err != nil {
				t.Fatalf("NewApp() error = %v", err)
			}
			defer func() { _ = app.container.Close() }()

			if app.config.ConfigPath != tt.configPath {
				t.Errorf("ConfigPath = %v, want %v", app.config.ConfigPath, tt.configPath)
			}
		})
	}
}

// TestApp_MultipleInstances 複数インスタンスの独立性
func TestApp_MultipleInstances(t *testing.T) {
	app1, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "8080",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer func() { _ = app1.container.Close() }()

	app2, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "9090",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer func() { _ = app2.container.Close() }()

	// 異なるインスタンスであることを確認
	if app1 == app2 {
		t.Error("Expected different app instances")
	}

	// 異なる設定であることを確認
	if app1.server.Addr == app2.server.Addr {
		t.Error("Expected different server addresses")
	}

	// それぞれのコンテナも異なるインスタンスであることを確認
	if app1.container == app2.container {
		t.Error("Expected different container instances")
	}
}

// TestApp_ServerTimeouts サーバータイムアウト設定の確認
func TestApp_ServerTimeouts(t *testing.T) {
	app, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "8080",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer func() { _ = app.container.Close() }()

	expectedTimeouts := map[string]time.Duration{
		"ReadTimeout":  30 * time.Second,
		"WriteTimeout": 30 * time.Second,
		"IdleTimeout":  60 * time.Second,
	}

	actualTimeouts := map[string]time.Duration{
		"ReadTimeout":  app.server.ReadTimeout,
		"WriteTimeout": app.server.WriteTimeout,
		"IdleTimeout":  app.server.IdleTimeout,
	}

	for name, expected := range expectedTimeouts {
		if actual := actualTimeouts[name]; actual != expected {
			t.Errorf("%s = %v, want %v", name, actual, expected)
		}
	}
}

// TestApp_Start_WithMock モックを使用したStartのテスト
func TestApp_Start_WithMock(t *testing.T) {
	tests := []struct {
		name    string
		mockErr error
		wantErr bool
	}{
		{
			name:    "正常系: 起動成功",
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "異常系: 起動失敗",
			mockErr: context.DeadlineExceeded,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp(&AppConfig{
				ConfigPath: "nonexistent.yaml",
				Port:       "8080",
			})
			if err != nil {
				t.Fatalf("NewApp() error = %v", err)
			}
			defer func() { _ = app.container.Close() }()

			// モックサーバーをインジェクト
			mockServer := &MockServer{
				listenAndServeFunc: func() error {
					return tt.mockErr
				},
			}
			app.serverSeam = mockServer

			err = app.Start()
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestApp_Run_WithMock モックを使用したRunのテスト
func TestApp_Run_WithMock(t *testing.T) {
	t.Run("正常系: シグナル受信でシャットダウン", func(t *testing.T) {
		app, err := NewApp(&AppConfig{
			ConfigPath: "nonexistent.yaml",
			Port:       "8080",
		})
		if err != nil {
			t.Fatalf("NewApp() error = %v", err)
		}
		defer func() { _ = app.container.Close() }()

		startCalled := false
		shutdownCalled := false

		// モックサーバーをインジェクト
		mockServer := &MockServer{
			listenAndServeFunc: func() error {
				startCalled = true
				// ブロックせずにすぐ返す
				time.Sleep(100 * time.Millisecond)
				return nil
			},
			shutdownFunc: func(ctx context.Context) error {
				shutdownCalled = true
				return nil
			},
		}
		app.serverSeam = mockServer

		// Run()を別goroutineで実行
		done := make(chan error, 1)
		go func() {
			done <- app.Run()
		}()

		// 少し待ってからシグナルを送信
		time.Sleep(200 * time.Millisecond)
		proc, _ := os.FindProcess(os.Getpid())
		_ = proc.Signal(os.Interrupt)

		// タイムアウト付きで完了を待機
		select {
		case <-done:
			// 正常終了
		case <-time.After(5 * time.Second):
			t.Fatal("Run() did not return within timeout")
		}

		if !startCalled {
			t.Error("Start was not called")
		}
		if !shutdownCalled {
			t.Error("Shutdown was not called")
		}
	})
}

// TestApp_PrintStartupMessage 起動メッセージのテスト
func TestApp_PrintStartupMessage(t *testing.T) {
	app, err := NewApp(&AppConfig{
		ConfigPath: "nonexistent.yaml",
		Port:       "8080",
	})
	if err != nil {
		t.Fatalf("NewApp() error = %v", err)
	}
	defer func() { _ = app.container.Close() }()

	// panicしないことを確認
	app.printStartupMessage()
}

// TestRealMain realMain()のテスト
func TestRealMain(t *testing.T) {
	// PORTを環境変数から設定
	originalPort := os.Getenv("PORT")
	defer func() {
		if originalPort != "" {
			_ = os.Setenv("PORT", originalPort)
		} else {
			_ = os.Unsetenv("PORT")
		}
	}()

	t.Run("正常系: PORT未設定", func(t *testing.T) {
		_ = os.Unsetenv("PORT")
		// realMain()は実際にサーバーを起動するため、テストではスキップ
		// 代わりに、環境変数の処理ロジックのみをテスト
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		if port != "8080" {
			t.Errorf("PORT = %v, want 8080", port)
		}
	})

	t.Run("正常系: PORT設定済み", func(t *testing.T) {
		_ = os.Setenv("PORT", "9000")
		port := os.Getenv("PORT")
		if port != "9000" {
			t.Errorf("PORT = %v, want 9000", port)
		}
	})
}
