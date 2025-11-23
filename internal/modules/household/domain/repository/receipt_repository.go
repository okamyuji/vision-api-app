package repository

import (
	"context"
	"time"

	"vision-api-app/internal/modules/household/domain/entity"
)

// ReceiptRepository レシートリポジトリのインターフェース
type ReceiptRepository interface {
	Create(ctx context.Context, receipt *entity.Receipt) error
	FindByID(ctx context.Context, id string) (*entity.Receipt, error)
	FindAll(ctx context.Context, limit, offset int) ([]*entity.Receipt, error)
	FindByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Receipt, error)
	Update(ctx context.Context, receipt *entity.Receipt) error
	Delete(ctx context.Context, id string) error
}

// ExpenseRepository 家計簿リポジトリのインターフェース
type ExpenseRepository interface {
	Create(ctx context.Context, entry *entity.ExpenseEntry) error
	FindByID(ctx context.Context, id string) (*entity.ExpenseEntry, error)
	FindAll(ctx context.Context, limit, offset int) ([]*entity.ExpenseEntry, error)
	FindByDateRange(ctx context.Context, start, end time.Time) ([]*entity.ExpenseEntry, error)
	FindByCategory(ctx context.Context, category string) ([]*entity.ExpenseEntry, error)
	Update(ctx context.Context, entry *entity.ExpenseEntry) error
	Delete(ctx context.Context, id string) error
}

// CategoryRepository カテゴリリポジトリのインターフェース
type CategoryRepository interface {
	Create(ctx context.Context, category *entity.Category) error
	FindByID(ctx context.Context, id string) (*entity.Category, error)
	FindAll(ctx context.Context) ([]*entity.Category, error)
	FindByName(ctx context.Context, name string) (*entity.Category, error)
	Update(ctx context.Context, category *entity.Category) error
	Delete(ctx context.Context, id string) error
}

// CacheRepository キャッシュリポジトリのインターフェース
type CacheRepository interface {
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
