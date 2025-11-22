package repository

import (
	"context"
	"time"
)

// CacheRepository キャッシュリポジトリのインターフェース
type CacheRepository interface {
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}
