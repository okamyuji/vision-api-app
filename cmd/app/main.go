package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"vision-api-app/internal/config"
	"vision-api-app/internal/presentation/di"
	"vision-api-app/internal/presentation/http/router"
)

// AppConfig アプリケーション設定
type AppConfig struct {
	ConfigPath string
	Port       string
}

// ServerInterface サーバーインターフェース（Seam化）
type ServerInterface interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// App アプリケーション構造体（Seamパターン）
type App struct {
	config     *AppConfig
	container  *di.Container
	server     *http.Server
	serverSeam ServerInterface // テスト用のSeam
}

// NewApp 新しいAppを作成
func NewApp(appCfg *AppConfig) (*App, error) {
	// ポートのデフォルト値設定
	if appCfg.Port == "" {
		appCfg.Port = "8080"
	}

	// 設定の読み込み
	cfg, err := config.Load(appCfg.ConfigPath)
	if err != nil {
		log.Printf("Failed to load config: %v. Using defaults.", err)
		cfg = config.DefaultConfig()
	}

	// DIコンテナの初期化
	container, err := di.NewContainer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize DI container: %w", err)
	}

	// ルーターの作成
	handler := router.NewRouter(container)

	// サーバーの設定
	server := &http.Server{
		Addr:         ":" + appCfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	app := &App{
		config:    appCfg,
		container: container,
		server:    server,
	}
	// デフォルトでは実際のサーバーを使用
	app.serverSeam = server

	return app, nil
}

// Start サーバーを起動
func (a *App) Start() error {
	// 起動メッセージ
	a.printStartupMessage()

	// サーバー起動（Seamを使用）
	return a.serverSeam.ListenAndServe()
}

// printStartupMessage 起動メッセージを出力
func (a *App) printStartupMessage() {
	fmt.Println("=== Vision API Server (Clean Architecture) ===")
	fmt.Printf("AI Provider: %s\n", a.container.AICorrectionUseCase().GetProviderName())
	fmt.Printf("Server listening on http://0.0.0.0:%s\n", a.config.Port)
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET  /health                      - Health check")
	fmt.Println("  POST /api/v1/vision/analyze       - Vision API (汎用OCR)")
	fmt.Println("  POST /api/v1/vision/receipt       - Receipt recognition (レシート認識)")
	fmt.Println("  POST /api/v1/vision/categorize    - Receipt categorization (カテゴリ判定)")
	fmt.Println()
}

// Shutdown サーバーをシャットダウン
func (a *App) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	// サーバーのシャットダウン（Seamを使用）
	if err := a.serverSeam.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	// コンテナのクローズ
	if err := a.container.Close(); err != nil {
		return fmt.Errorf("container close failed: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

// Run アプリケーションを実行（グレースフルシャットダウン付き）
func (a *App) Run() error {
	// サーバー起動（goroutine）
	serverErr := make(chan error, 1)
	go func() {
		if err := a.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// シグナルの待機
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server failed: %w", err)
	case <-quit:
		// グレースフルシャットダウン
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		return a.Shutdown(ctx)
	}
}

// realMain 実際のmain処理（テスト可能にするため分離）
func realMain() error {
	// ホームディレクトリの取得
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Failed to get home directory: %v. Using current directory.", err)
		homeDir = "."
	}

	configPath := filepath.Join(homeDir, ".tesseract-ocr-app", "config.yaml")

	// ポート番号の取得
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// アプリケーション設定
	appCfg := &AppConfig{
		ConfigPath: configPath,
		Port:       port,
	}

	// アプリケーションの作成
	app, err := NewApp(appCfg)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}

	// アプリケーションの実行
	return app.Run()
}

func main() {
	if err := realMain(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}
