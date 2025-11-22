package handler

import (
	"context"
	"time"

	"vision-api-app/internal/domain/entity"
)

// AICorrectionUseCaseInterface はAI補正ユースケースのインターフェース
type AICorrectionUseCaseInterface interface {
	Correct(text string) (*entity.AIResult, error)
	RecognizeImage(imageData []byte) (*entity.AIResult, error)
	RecognizeReceipt(imageData []byte) (*entity.AIResult, error)
	CategorizeReceipt(receiptInfo string) (*entity.AIResult, error)
	GetProviderName() string
}

// CacheRepositoryInterface はキャッシュリポジトリのインターフェース
type CacheRepositoryInterface interface {
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// ReceiptRepositoryInterface はレシートリポジトリのインターフェース
type ReceiptRepositoryInterface interface {
	Create(ctx context.Context, receipt *entity.Receipt) error
	FindByID(ctx context.Context, id string) (*entity.Receipt, error)
}
