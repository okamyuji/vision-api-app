package di

import (
	"fmt"
	"vision-api-app/internal/config"
	"vision-api-app/internal/domain/repository"
	"vision-api-app/internal/infrastructure/ai"
	"vision-api-app/internal/infrastructure/cache"
	"vision-api-app/internal/infrastructure/database"
	"vision-api-app/internal/usecase"
)

// Container DIコンテナ
type Container struct {
	config *config.Config

	// Repositories
	aiRepo      repository.AIRepository
	cacheRepo   repository.CacheRepository
	receiptRepo repository.ReceiptRepository

	// UseCases
	aiCorrectionUseCase *usecase.AICorrectionUseCase
}

// NewContainer 新しいDIコンテナを作成
func NewContainer(cfg *config.Config) (*Container, error) {
	c := &Container{
		config: cfg,
	}

	// Repositoriesの初期化
	c.aiRepo = ai.NewClaudeRepository(&cfg.Anthropic)

	// Redis Repository
	redisRepo, err := cache.NewRedisRepository(&cfg.Redis)
	if err != nil {
		// Redisが利用できない場合はnilのまま（オプショナル）
		fmt.Printf("Warning: Redis not available: %v\n", err)
		c.cacheRepo = nil
	} else {
		fmt.Printf("✅ Redis connected successfully\n")
		c.cacheRepo = redisRepo
	}

	// Receipt Repository
	receiptRepo, err := database.NewBunReceiptRepository(&cfg.MySQL)
	if err != nil {
		// MySQLが利用できない場合はnilのまま（オプショナル）
		fmt.Printf("Warning: MySQL not available: %v\n", err)
		c.receiptRepo = nil
	} else {
		fmt.Printf("✅ MySQL connected successfully\n")
		c.receiptRepo = receiptRepo
	}

	// UseCasesの初期化（DIによる依存性注入）
	c.aiCorrectionUseCase = usecase.NewAICorrectionUseCase(c.aiRepo)

	return c, nil
}

// Config 設定を返す
func (c *Container) Config() *config.Config {
	return c.config
}

// AICorrectionUseCase AI補正ユースケースを返す
func (c *Container) AICorrectionUseCase() *usecase.AICorrectionUseCase {
	return c.aiCorrectionUseCase
}

// CacheRepository キャッシュリポジトリを返す
func (c *Container) CacheRepository() repository.CacheRepository {
	return c.cacheRepo
}

// ReceiptRepository レシートリポジトリを返す
func (c *Container) ReceiptRepository() repository.ReceiptRepository {
	return c.receiptRepo
}

// Close リソースを解放
func (c *Container) Close() error {
	// Redis接続を閉じる
	if c.cacheRepo != nil {
		if closer, ok := c.cacheRepo.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}

	// MySQL接続を閉じる
	if c.receiptRepo != nil {
		if closer, ok := c.receiptRepo.(interface{ Close() error }); ok {
			_ = closer.Close()
		}
	}

	return nil
}
