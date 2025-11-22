package cache

import (
	"context"
	"fmt"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"

	"vision-api-app/internal/config"
)

var (
	redisContainer     testcontainers.Container
	redisContainerOnce sync.Once
	redisContainerErr  error
	redisHost          string
	redisPort          string
	redisMu            sync.Mutex
)

// RedisTestContainer Redisテストコンテナの情報
type RedisTestContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
}

// GetOrCreateRedisTestContainer Redisテストコンテナを取得または作成（シングルトン）
func GetOrCreateRedisTestContainer(ctx context.Context) (*RedisTestContainer, error) {
	redisContainerOnce.Do(func() {
		// Redisコンテナの起動
		container, err := redis.Run(ctx, "redis:7-alpine")
		if err != nil {
			redisContainerErr = fmt.Errorf("failed to start redis container: %w", err)
			return
		}

		redisContainer = container

		// 接続情報の取得
		host, err := container.Host(ctx)
		if err != nil {
			redisContainerErr = fmt.Errorf("failed to get container host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "6379")
		if err != nil {
			redisContainerErr = fmt.Errorf("failed to get container port: %w", err)
			return
		}

		redisHost = host
		redisPort = port.Port()
	})

	if redisContainerErr != nil {
		return nil, redisContainerErr
	}

	return &RedisTestContainer{
		Container: redisContainer,
		Host:      redisHost,
		Port:      redisPort,
	}, nil
}

// NewTestRedisRepository テスト用のRedisRepositoryを作成
func NewTestRedisRepository(ctx context.Context) (*RedisRepository, error) {
	tc, err := GetOrCreateRedisTestContainer(ctx)
	if err != nil {
		return nil, err
	}

	cfg := &config.RedisConfig{
		Host:     tc.Host,
		Port:     mustAtoi(tc.Port),
		Password: "",
		DB:       0,
	}

	return NewRedisRepository(cfg)
}

// CleanupRedis Redisのデータをクリーンアップ
func CleanupRedis(ctx context.Context, repo *RedisRepository) error {
	redisMu.Lock()
	defer redisMu.Unlock()

	// 全てのキーを削除
	return repo.client.FlushDB(ctx).Err()
}

// CloseRedisTestContainer Redisテストコンテナを終了
func CloseRedisTestContainer(ctx context.Context) error {
	if redisContainer != nil {
		return redisContainer.Terminate(ctx)
	}
	return nil
}

// mustAtoi stringをintに変換（エラー時は0）
func mustAtoi(s string) int {
	var result int
	_, _ = fmt.Sscanf(s, "%d", &result)
	return result
}
