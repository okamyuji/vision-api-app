package cache

import (
	"context"
	"fmt"
	"testing"
	"time"

	"vision-api-app/internal/config"
	"vision-api-app/internal/modules/shared/infrastructure/testcontainer"
)

func setupRedisRepo(t *testing.T) (*RedisRepository, func()) {
	t.Helper()
	ctx := context.Background()

	// TestContainer起動
	redisContainer, err := testcontainer.StartRedis(ctx, t)
	if err != nil {
		t.Fatalf("Failed to start redis container: %v", err)
	}

	// Redisリポジトリ作成
	port := 6379
	if _, err := fmt.Sscanf(redisContainer.Port, "%d", &port); err != nil {
		_ = redisContainer.Close(ctx)
		t.Fatalf("Failed to parse redis port: %v", err)
	}
	repo, err := NewRedisRepository(&config.RedisConfig{
		Host:     redisContainer.Host,
		Port:     port,
		Password: "",
		DB:       0,
	})
	if err != nil {
		_ = redisContainer.Close(ctx)
		t.Fatalf("Failed to create redis repository: %v", err)
	}

	return repo, func() {
		_ = repo.Close()
		_ = redisContainer.Close(ctx)
	}
}

func TestRedisRepository_Set(t *testing.T) {
	repo, cleanup := setupRedisRepo(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name       string
		key        string
		value      []byte
		expiration time.Duration
		wantErr    bool
	}{
		{
			name:       "正常系: 通常のSet",
			key:        "test:key1",
			value:      []byte("value1"),
			expiration: 1 * time.Hour,
			wantErr:    false,
		},
		{
			name:       "正常系: 短い有効期限",
			key:        "test:key2",
			value:      []byte("value2"),
			expiration: 1 * time.Second,
			wantErr:    false,
		},
		{
			name:       "正常系: 長い値",
			key:        "test:key3",
			value:      make([]byte, 10000),
			expiration: 1 * time.Hour,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Set(ctx, tt.key, tt.value, tt.expiration)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRedisRepository_Get(t *testing.T) {
	repo, cleanup := setupRedisRepo(t)
	defer cleanup()

	ctx := context.Background()

	// テストデータをセット
	testKey := "test:get:key"
	testValue := []byte("test_value")
	if err := repo.Set(ctx, testKey, testValue, 1*time.Hour); err != nil {
		t.Fatalf("Failed to set test data: %v", err)
	}

	tests := []struct {
		name      string
		key       string
		wantValue []byte
		wantErr   bool
	}{
		{
			name:      "正常系: 存在するキー",
			key:       testKey,
			wantValue: testValue,
			wantErr:   false,
		},
		{
			name:      "異常系: 存在しないキー",
			key:       "test:nonexistent",
			wantValue: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := repo.Get(ctx, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(value) != string(tt.wantValue) {
				t.Errorf("Get() value = %v, want %v", string(value), string(tt.wantValue))
			}
		})
	}
}

func TestRedisRepository_Delete(t *testing.T) {
	repo, cleanup := setupRedisRepo(t)
	defer cleanup()

	ctx := context.Background()

	// テストデータをセット
	testKey := "test:delete:key"
	if err := repo.Set(ctx, testKey, []byte("value"), 1*time.Hour); err != nil {
		t.Fatalf("Failed to set test data: %v", err)
	}

	// 削除
	if err := repo.Delete(ctx, testKey); err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// 削除されたことを確認
	_, err := repo.Get(ctx, testKey)
	if err == nil {
		t.Error("Key still exists after Delete()")
	}
}

func TestRedisRepository_Exists(t *testing.T) {
	repo, cleanup := setupRedisRepo(t)
	defer cleanup()

	ctx := context.Background()

	// テストデータをセット
	testKey := "test:exists:key"
	if err := repo.Set(ctx, testKey, []byte("value"), 1*time.Hour); err != nil {
		t.Fatalf("Failed to set test data: %v", err)
	}

	tests := []struct {
		name       string
		key        string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "正常系: 存在するキー",
			key:        testKey,
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "正常系: 存在しないキー",
			key:        "test:nonexistent",
			wantExists: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := repo.Exists(ctx, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if exists != tt.wantExists {
				t.Errorf("Exists() = %v, want %v", exists, tt.wantExists)
			}
		})
	}
}
