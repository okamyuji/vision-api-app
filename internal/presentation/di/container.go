package di

import (
	"fmt"

	"vision-api-app/internal/config"
	householdHandler "vision-api-app/internal/modules/household/presentation/handler"
	householdUsecase "vision-api-app/internal/modules/household/usecase"
	sharedAI "vision-api-app/internal/modules/shared/infrastructure/ai"
	sharedCache "vision-api-app/internal/modules/shared/infrastructure/cache"
	sharedDB "vision-api-app/internal/modules/shared/infrastructure/database"
	visionHandler "vision-api-app/internal/modules/vision/presentation/handler"
	visionUsecase "vision-api-app/internal/modules/vision/usecase"
)

// Container DIコンテナ
type Container struct {
	// Shared Infrastructure
	aiRepo      *sharedAI.ClaudeRepository
	cacheRepo   *sharedCache.RedisRepository
	receiptRepo *sharedDB.BunReceiptRepository
	expenseRepo *sharedDB.BunExpenseRepository

	// Vision Module
	aiCorrectionUseCase *visionUsecase.AICorrectionUseCase
	visionHandler       *visionHandler.VisionHandler

	// Household Module
	receiptUseCase   *householdUsecase.ReceiptUseCase
	householdUseCase *householdUsecase.HouseholdUseCase
	webHandler       *householdHandler.WebHandler
}

// NewContainer 新しいContainerを作成
func NewContainer(cfg *config.Config) (*Container, error) {
	container := &Container{}

	// Shared Infrastructure: AI Repository
	aiRepo := sharedAI.NewClaudeRepository(&cfg.Anthropic)
	container.aiRepo = aiRepo

	// Shared Infrastructure: Cache Repository
	cacheRepo, err := sharedCache.NewRedisRepository(&cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache repository: %w", err)
	}
	container.cacheRepo = cacheRepo

	// Shared Infrastructure: Receipt Repository
	receiptRepo, err := sharedDB.NewBunReceiptRepository(&cfg.MySQL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize receipt repository: %w", err)
	}
	container.receiptRepo = receiptRepo

	// Shared Infrastructure: Expense Repository
	expenseRepo, err := sharedDB.NewBunExpenseRepository(&cfg.MySQL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize expense repository: %w", err)
	}
	container.expenseRepo = expenseRepo

	// Vision Module: UseCase
	aiCorrectionUseCase := visionUsecase.NewAICorrectionUseCase(aiRepo)
	container.aiCorrectionUseCase = aiCorrectionUseCase

	// Vision Module: Handler
	visionHandler := visionHandler.NewVisionHandler(aiCorrectionUseCase, cacheRepo)
	container.visionHandler = visionHandler

	// Household Module: Receipt UseCase
	receiptUseCase := householdUsecase.NewReceiptUseCase(aiRepo, receiptRepo)
	container.receiptUseCase = receiptUseCase

	// Household Module: Household UseCase
	householdUseCase := householdUsecase.NewHouseholdUseCase(receiptRepo, expenseRepo)
	container.householdUseCase = householdUseCase

	// Household Module: Web Handler
	webHandler, err := householdHandler.NewWebHandler(receiptUseCase, householdUseCase)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize web handler: %w", err)
	}
	container.webHandler = webHandler

	return container, nil
}

// AICorrectionUseCase Vision AI補正ユースケースを取得
func (c *Container) AICorrectionUseCase() *visionUsecase.AICorrectionUseCase {
	return c.aiCorrectionUseCase
}

// VisionHandler Vision APIハンドラーを取得
func (c *Container) VisionHandler() *visionHandler.VisionHandler {
	return c.visionHandler
}

// WebHandler Web UIハンドラーを取得
func (c *Container) WebHandler() *householdHandler.WebHandler {
	return c.webHandler
}

// Close リソースをクローズ
func (c *Container) Close() error {
	if c.cacheRepo != nil {
		if err := c.cacheRepo.Close(); err != nil {
			return fmt.Errorf("failed to close cache repository: %w", err)
		}
	}

	if c.receiptRepo != nil {
		if err := c.receiptRepo.Close(); err != nil {
			return fmt.Errorf("failed to close receipt repository: %w", err)
		}
	}

	if c.expenseRepo != nil {
		if err := c.expenseRepo.Close(); err != nil {
			return fmt.Errorf("failed to close expense repository: %w", err)
		}
	}

	return nil
}
