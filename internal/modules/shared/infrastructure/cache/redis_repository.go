package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"vision-api-app/internal/config"
)

// RedisRepository Redis実装
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository 新しいRedisRepositoryを作成
func NewRedisRepository(cfg *config.RedisConfig) (*RedisRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 接続確認
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisRepository{client: client}, nil
}

// Set キーと値を設定
func (r *RedisRepository) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if err := r.client.Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

// Get キーから値を取得
func (r *RedisRepository) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("cache not found: %s", key)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}
	return val, nil
}

// Delete キーを削除
func (r *RedisRepository) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}
	return nil
}

// Exists キーが存在するか確認
func (r *RedisRepository) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check cache existence: %w", err)
	}
	return count > 0, nil
}

// Close Redis接続を閉じる
func (r *RedisRepository) Close() error {
	return r.client.Close()
}
