package cache

import (
	"context"
	"testing"
	"time"
	"vision-api-app/internal/config"
)

func TestRedisRepository_Set_Get(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("正常系_値の設定と取得", func(t *testing.T) {
		key := "test-key-1"
		value := []byte("test value")

		// Set
		if err := repo.Set(ctx, key, value, 10*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Get
		got, err := repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if string(got) != string(value) {
			t.Errorf("Get() = %v, want %v", string(got), string(value))
		}
	})

	t.Run("正常系_有効期限付き", func(t *testing.T) {
		key := "test-key-expire"
		value := []byte("expire test")

		// 1秒の有効期限で設定
		if err := repo.Set(ctx, key, value, 1*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// すぐに取得できる
		_, err := repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		// 2秒待つ
		time.Sleep(2 * time.Second)

		// 有効期限切れで取得できない
		_, err = repo.Get(ctx, key)
		if err == nil {
			t.Error("Expected error for expired key, got nil")
		}
	})

	t.Run("異常系_存在しないキー", func(t *testing.T) {
		_, err := repo.Get(ctx, "non-existent-key")
		if err == nil {
			t.Error("Expected error for non-existent key, got nil")
		}
	})
}

func TestRedisRepository_Delete(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("正常系_削除", func(t *testing.T) {
		key := "test-key-delete"
		value := []byte("to be deleted")

		// Set
		if err := repo.Set(ctx, key, value, 10*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Delete
		if err := repo.Delete(ctx, key); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// Get（削除後なのでエラー）
		_, err := repo.Get(ctx, key)
		if err == nil {
			t.Error("Expected error for deleted key, got nil")
		}
	})

	t.Run("正常系_存在しないキーの削除", func(t *testing.T) {
		// 存在しないキーを削除してもエラーにならない
		err := repo.Delete(ctx, "non-existent-key-delete")
		if err != nil {
			t.Errorf("Delete() error = %v, want nil", err)
		}
	})

	t.Run("正常系_存在しないキーの削除", func(t *testing.T) {
		// 存在しないキーの削除はエラーにならない
		if err := repo.Delete(ctx, "non-existent-key"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
	})
}

func TestRedisRepository_Exists(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("正常系_存在確認", func(t *testing.T) {
		key := "test-key-exists"
		value := []byte("exists test")

		// 最初は存在しない
		exists, err := repo.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Key should not exist")
		}

		// Set
		if err := repo.Set(ctx, key, value, 10*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// 存在する
		exists, err = repo.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if !exists {
			t.Error("Key should exist")
		}

		// Delete
		if err := repo.Delete(ctx, key); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// 存在しない
		exists, err = repo.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Exists() error = %v", err)
		}
		if exists {
			t.Error("Key should not exist after deletion")
		}
	})
}

func TestRedisRepository_LargeValue(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("境界値_大きな値", func(t *testing.T) {
		key := "test-key-large"
		// 1MBのデータ
		value := make([]byte, 1024*1024)
		for i := range value {
			value[i] = byte(i % 256)
		}

		// Set
		if err := repo.Set(ctx, key, value, 10*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Get
		got, err := repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if len(got) != len(value) {
			t.Errorf("Get() length = %v, want %v", len(got), len(value))
		}

		// データの整合性確認（先頭100バイト）
		for i := 0; i < 100; i++ {
			if got[i] != value[i] {
				t.Errorf("Data mismatch at index %d: got %v, want %v", i, got[i], value[i])
				break
			}
		}
	})
}

func TestRedisRepository_EmptyValue(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("エッジケース_空の値", func(t *testing.T) {
		key := "test-key-empty"
		value := []byte("")

		// Set
		if err := repo.Set(ctx, key, value, 10*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Get
		got, err := repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if len(got) != 0 {
			t.Errorf("Get() length = %v, want 0", len(got))
		}
	})
}

func TestRedisRepository_MultipleKeys(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("正常系_複数キーの操作", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3", "key4", "key5"}
		values := [][]byte{
			[]byte("value1"),
			[]byte("value2"),
			[]byte("value3"),
			[]byte("value4"),
			[]byte("value5"),
		}

		// 全てSet
		for i, key := range keys {
			if err := repo.Set(ctx, key, values[i], 10*time.Second); err != nil {
				t.Fatalf("Set() error = %v", err)
			}
		}

		// 全てGet
		for i, key := range keys {
			got, err := repo.Get(ctx, key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if string(got) != string(values[i]) {
				t.Errorf("Get(%s) = %v, want %v", key, string(got), string(values[i]))
			}
		}

		// 一部削除
		if err := repo.Delete(ctx, keys[0]); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if err := repo.Delete(ctx, keys[2]); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		// 存在確認
		for i, key := range keys {
			exists, err := repo.Exists(ctx, key)
			if err != nil {
				t.Fatalf("Exists() error = %v", err)
			}

			shouldExist := i != 0 && i != 2
			if exists != shouldExist {
				t.Errorf("Exists(%s) = %v, want %v", key, exists, shouldExist)
			}
		}
	})
}

func TestRedisRepository_NoExpiration(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("正常系_有効期限なし", func(t *testing.T) {
		key := "test-key-no-expire"
		value := []byte("no expiration")

		// 有効期限0（永続化）
		if err := repo.Set(ctx, key, value, 0); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// すぐに取得
		got, err := repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Get() = %v, want %v", string(got), string(value))
		}

		// 3秒待っても取得できる
		time.Sleep(3 * time.Second)
		got, err = repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value) {
			t.Errorf("Get() after wait = %v, want %v", string(got), string(value))
		}
	})
}

func TestRedisRepository_SpecialCharacters(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("エッジケース_特殊文字", func(t *testing.T) {
		tests := []struct {
			name  string
			key   string
			value []byte
		}{
			{"日本語キー", "テストキー", []byte("日本語の値")},
			{"特殊文字", "key:with:colons", []byte("special!@#$%^&*()")},
			{"スペース", "key with spaces", []byte("value with spaces")},
			{"JSON", "json-key", []byte(`{"name":"test","value":123}`)},
			{"改行", "newline-key", []byte("line1\nline2\nline3")},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if err := repo.Set(ctx, tt.key, tt.value, 10*time.Second); err != nil {
					t.Fatalf("Set() error = %v", err)
				}

				got, err := repo.Get(ctx, tt.key)
				if err != nil {
					t.Fatalf("Get() error = %v", err)
				}

				if string(got) != string(tt.value) {
					t.Errorf("Get() = %v, want %v", string(got), string(tt.value))
				}
			})
		}
	})
}

func TestRedisRepository_Overwrite(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("正常系_値の上書き", func(t *testing.T) {
		key := "test-key-overwrite"
		value1 := []byte("original value")
		value2 := []byte("updated value")

		// 最初の値をSet
		if err := repo.Set(ctx, key, value1, 10*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// 取得確認
		got, err := repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value1) {
			t.Errorf("Get() = %v, want %v", string(got), string(value1))
		}

		// 上書き
		if err := repo.Set(ctx, key, value2, 10*time.Second); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// 新しい値を取得
		got, err = repo.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value2) {
			t.Errorf("Get() = %v, want %v", string(got), string(value2))
		}
	})
}

func TestRedisRepository_ContextCancellation(t *testing.T) {
	baseCtx := context.Background()
	repo, err := NewTestRedisRepository(baseCtx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(baseCtx, repo) }()

	t.Run("異常系_キャンセルされたコンテキスト_Set", func(t *testing.T) {
		ctx, cancel := context.WithCancel(baseCtx)
		cancel() // すぐにキャンセル

		err := repo.Set(ctx, "test-key", []byte("test"), 10*time.Second)
		if err == nil {
			t.Error("Expected error for cancelled context in Set, got nil")
		}
	})

	t.Run("異常系_キャンセルされたコンテキスト_Get", func(t *testing.T) {
		// 先にキーを設定
		_ = repo.Set(baseCtx, "test-key-get-cancel", []byte("test"), 10*time.Second)

		ctx, cancel := context.WithCancel(baseCtx)
		cancel() // すぐにキャンセル

		_, err := repo.Get(ctx, "test-key-get-cancel")
		if err == nil {
			t.Error("Expected error for cancelled context in Get, got nil")
		}
	})

	t.Run("異常系_キャンセルされたコンテキスト_Delete", func(t *testing.T) {
		ctx, cancel := context.WithCancel(baseCtx)
		cancel() // すぐにキャンセル

		err := repo.Delete(ctx, "test-key")
		if err == nil {
			t.Error("Expected error for cancelled context in Delete, got nil")
		}
	})

	t.Run("異常系_キャンセルされたコンテキスト_Exists", func(t *testing.T) {
		ctx, cancel := context.WithCancel(baseCtx)
		cancel() // すぐにキャンセル

		_, err := repo.Exists(ctx, "test-key")
		if err == nil {
			t.Error("Expected error for cancelled context in Exists, got nil")
		}
	})
}

func TestRedisRepository_EdgeCases(t *testing.T) {
	ctx := context.Background()
	repo, err := NewTestRedisRepository(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = repo.Close() }()
	defer func() { _ = CleanupRedis(ctx, repo) }()

	t.Run("エッジケース_非常に長いキー", func(t *testing.T) {
		// 10KBのキー名
		longKey := string(make([]byte, 10*1024))
		value := []byte("test value")

		err := repo.Set(ctx, longKey, value, 10*time.Second)
		if err != nil {
			t.Logf("Expected behavior: Set with very long key failed: %v", err)
		}
	})

	t.Run("エッジケース_負のExpiration", func(t *testing.T) {
		key := "test-key-negative-exp"
		value := []byte("test")

		// 負の有効期限（即座に期限切れ）
		err := repo.Set(ctx, key, value, -1*time.Second)
		if err != nil {
			t.Logf("Set with negative expiration may fail: %v", err)
		}

		// 取得しようとする（期限切れまたは存在しない）
		_, err = repo.Get(ctx, key)
		if err == nil {
			t.Log("Key may not be retrievable with negative expiration")
		}
	})
}

func TestNewRedisRepository_ConnectionFailure(t *testing.T) {
	t.Run("異常系_接続失敗", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "nonexistent-host-12345",
			Port:     6379,
			Password: "",
			DB:       0,
		}

		_, err := NewRedisRepository(cfg)
		if err == nil {
			t.Error("Expected error for connection failure, got nil")
		}
	})

	t.Run("異常系_無効なポート", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "localhost",
			Port:     1, // 無効なポート
			Password: "",
			DB:       0,
		}

		_, err := NewRedisRepository(cfg)
		if err == nil {
			t.Error("Expected error for invalid port, got nil")
		}
	})
}
