package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"

	_ "github.com/go-sql-driver/mysql"
)

var (
	testContainer     testcontainers.Container
	testContainerOnce sync.Once
	testContainerErr  error
	testDSN           string
	testMu            sync.Mutex
)

// TestContainer テストコンテナの情報
type TestContainer struct {
	Container testcontainers.Container
	DSN       string
}

// GetOrCreateTestContainer テストコンテナを取得または作成（シングルトン）
func GetOrCreateTestContainer(ctx context.Context) (*TestContainer, error) {
	testContainerOnce.Do(func() {
		// MySQLコンテナの起動
		container, err := mysql.Run(ctx,
			"mysql:8.0",
			mysql.WithDatabase("household"),
			mysql.WithUsername("root"),
			mysql.WithPassword("testpass"),
			mysql.WithScripts("../../../scripts/init.sql"),
		)
		if err != nil {
			testContainerErr = fmt.Errorf("failed to start mysql container: %w", err)
			return
		}

		testContainer = container

		// 接続文字列の取得
		host, err := container.Host(ctx)
		if err != nil {
			testContainerErr = fmt.Errorf("failed to get container host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "3306")
		if err != nil {
			testContainerErr = fmt.Errorf("failed to get container port: %w", err)
			return
		}

		testDSN = fmt.Sprintf("root:testpass@tcp(%s:%s)/household?charset=utf8mb4&parseTime=true&loc=Local",
			host, port.Port())

		// mysql.WithScripts()でinit.sqlが自動実行されるため、追加の初期化は不要
	})

	if testContainerErr != nil {
		return nil, testContainerErr
	}

	return &TestContainer{
		Container: testContainer,
		DSN:       testDSN,
	}, nil
}

// NewTestDB テスト用のDBインスタンスを作成
func NewTestDB(ctx context.Context) (*bun.DB, error) {
	tc, err := GetOrCreateTestContainer(ctx)
	if err != nil {
		return nil, err
	}

	sqldb, err := sql.Open("mysql", tc.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := bun.NewDB(sqldb, mysqldialect.New())

	return db, nil
}

// CleanupTestTables テストテーブルをクリーンアップ
func CleanupTestTables(ctx context.Context, db *bun.DB) error {
	testMu.Lock()
	defer testMu.Unlock()

	// 外部キー制約を一時的に無効化
	if _, err := db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=0"); err != nil {
		return err
	}

	// テーブルをTRUNCATE
	tables := []string{
		"receipt_items",
		"expense_entries",
		"receipts",
		// categoriesはマスタデータなので削除しない
	}

	for _, table := range tables {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
			return err
		}
	}

	// 外部キー制約を再度有効化
	if _, err := db.ExecContext(ctx, "SET FOREIGN_KEY_CHECKS=1"); err != nil {
		return err
	}

	return nil
}

// CloseTestContainer テストコンテナを終了（テスト終了時に呼ぶ）
func CloseTestContainer(ctx context.Context) error {
	if testContainer != nil {
		return testContainer.Terminate(ctx)
	}
	return nil
}
