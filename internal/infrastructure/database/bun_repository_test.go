package database

import (
	"context"
	"testing"
	"time"

	"vision-api-app/internal/config"
	"vision-api-app/internal/domain/entity"

	"github.com/google/uuid"
)

// =============================================================================
// BunReceiptRepository Tests
// =============================================================================

func TestNewBunReceiptRepository(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.MySQLConfig
		wantErr bool
	}{
		{
			name: "異常系: 無効なホスト",
			cfg: &config.MySQLConfig{
				Host:     "invalid-host-that-does-not-exist",
				Port:     3306,
				User:     "root",
				Password: "password",
				Database: "test",
			},
			wantErr: true,
		},
		{
			name: "異常系: 無効なポート",
			cfg: &config.MySQLConfig{
				Host:     "localhost",
				Port:     99999,
				User:     "root",
				Password: "password",
				Database: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewBunReceiptRepository(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBunReceiptRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && repo == nil {
				t.Error("Expected non-nil repository")
			}

			if repo != nil {
				_ = repo.Close()
			}
		})
	}
}

func TestBunReceiptRepository_Create(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewBunReceiptRepositoryWithDB(db)

	t.Run("正常系_レシート作成", func(t *testing.T) {
		defer func() { _ = CleanupTestTables(ctx, db) }()

		receipt := entity.NewReceipt(
			uuid.NewString(),
			"テストストア",
			time.Now(),
			1000,
			100,
			"食費",
		)

		if err := repo.Create(ctx, receipt); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// 検証
		found, err := repo.FindByID(ctx, receipt.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.StoreName != receipt.StoreName {
			t.Errorf("StoreName = %v, want %v", found.StoreName, receipt.StoreName)
		}
	})

	t.Run("正常系_明細付きレシート作成", func(t *testing.T) {
		defer func() { _ = CleanupTestTables(ctx, db) }()

		receipt := entity.NewReceipt(
			uuid.NewString(),
			"スーパーマーケット",
			time.Now(),
			500,
			50,
			"食費",
		)

		item := entity.NewReceiptItem(
			uuid.NewString(),
			receipt.ID,
			"りんご",
			3,
			150,
		)
		receipt.AddItem(item)

		if err := repo.Create(ctx, receipt); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := repo.FindByID(ctx, receipt.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if len(found.Items) != 1 {
			t.Errorf("Items count = %v, want 1", len(found.Items))
		}

		if found.Items[0].Name != "りんご" {
			t.Errorf("Item name = %v, want りんご", found.Items[0].Name)
		}
	})

	t.Run("異常系_空のストア名", func(t *testing.T) {
		defer func() { _ = CleanupTestTables(ctx, db) }()

		receipt := entity.NewReceipt(
			uuid.NewString(),
			"",
			time.Now(),
			1000,
			100,
			"食費",
		)

		if !receipt.IsValid() {
			return // 期待通り無効
		}
		t.Error("Expected invalid receipt")
	})
}

func TestBunReceiptRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunReceiptRepositoryWithDB(db)

	// テストデータ作成
	for i := 0; i < 5; i++ {
		receipt := entity.NewReceipt(
			uuid.NewString(),
			"Store "+string(rune('A'+i)),
			time.Now().Add(time.Duration(i)*time.Hour),
			1000+i*100,
			100+i*10,
			"食費",
		)
		if err := repo.Create(ctx, receipt); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	tests := []struct {
		name      string
		limit     int
		offset    int
		wantCount int
	}{
		{
			name:      "正常系: 全件取得",
			limit:     10,
			offset:    0,
			wantCount: 5,
		},
		{
			name:      "正常系: limit=3",
			limit:     3,
			offset:    0,
			wantCount: 3,
		},
		{
			name:      "正常系: offset=2",
			limit:     10,
			offset:    2,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipts, err := repo.FindAll(ctx, tt.limit, tt.offset)
			if err != nil {
				t.Fatalf("FindAll() error = %v", err)
			}

			if len(receipts) != tt.wantCount {
				t.Errorf("FindAll() count = %d, want %d", len(receipts), tt.wantCount)
			}
		})
	}
}

func TestBunReceiptRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunReceiptRepositoryWithDB(db)

	_, err = repo.FindByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent ID, got nil")
	}
}

func TestBunReceiptRepository_FindByDateRange(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunReceiptRepositoryWithDB(db)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tomorrow := now.AddDate(0, 0, 1)

	// テストデータ作成
	receipts := []*entity.Receipt{
		entity.NewReceipt(uuid.NewString(), "店1", yesterday, 100, 10, "食費"),
		entity.NewReceipt(uuid.NewString(), "店2", now, 200, 20, "食費"),
		entity.NewReceipt(uuid.NewString(), "店3", tomorrow, 300, 30, "食費"),
	}

	for _, r := range receipts {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// 日付範囲検索（yesterday <= date <= now）
	start := yesterday.Truncate(24 * time.Hour)
	end := now.Add(24 * time.Hour).Truncate(24 * time.Hour)

	found, err := repo.FindByDateRange(ctx, start, end)
	if err != nil {
		t.Fatalf("FindByDateRange() error = %v", err)
	}

	// yesterday と now のレシートが含まれる（tomorrowは除外）
	if len(found) < 2 {
		t.Errorf("Found count = %v, want at least 2", len(found))
	}
}

func TestBunReceiptRepository_Update_Delete(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunReceiptRepositoryWithDB(db)

	receipt := entity.NewReceipt(
		uuid.NewString(),
		"元のストア名",
		time.Now(),
		1000,
		100,
		"食費",
	)

	if err := repo.Create(ctx, receipt); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 更新
	receipt.StoreName = "新しいストア名"
	if err := repo.Update(ctx, receipt); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, err := repo.FindByID(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if updated.StoreName != "新しいストア名" {
		t.Errorf("StoreName = %v, want 新しいストア名", updated.StoreName)
	}

	// 削除
	if err := repo.Delete(ctx, receipt.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = repo.FindByID(ctx, receipt.ID)
	if err == nil {
		t.Error("Expected error for deleted receipt")
	}
}

func TestBunReceiptRepository_Close(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	repo := NewBunReceiptRepositoryWithDB(db)

	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// 2回目のCloseも問題ないことを確認
	if err := repo.Close(); err != nil {
		t.Errorf("Second Close() error = %v", err)
	}
}

// =============================================================================
// BunExpenseRepository Tests
// =============================================================================

func TestNewBunExpenseRepository(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.MySQLConfig
		wantErr bool
	}{
		{
			name: "異常系: 無効なホスト",
			cfg: &config.MySQLConfig{
				Host:     "invalid-host-that-does-not-exist",
				Port:     3306,
				User:     "root",
				Password: "password",
				Database: "test",
			},
			wantErr: true,
		},
		{
			name: "異常系: 無効なポート",
			cfg: &config.MySQLConfig{
				Host:     "localhost",
				Port:     99999,
				User:     "root",
				Password: "password",
				Database: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewBunExpenseRepository(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBunExpenseRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && repo == nil {
				t.Error("Expected non-nil repository")
			}

			if repo != nil {
				_ = repo.Close()
			}
		})
	}
}

func TestBunExpenseRepository_Create(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	t.Run("正常系_家計簿エントリ作成", func(t *testing.T) {
		entry := entity.NewExpenseEntry(
			uuid.NewString(),
			time.Now(),
			"食費",
			1500,
			"ランチ代",
			[]string{"外食", "平日"},
		)

		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// 直接DBから確認
		var raw ExpenseEntry
		err := db.NewSelect().Model(&raw).Where("id = ?", entry.ID).Scan(ctx)
		if err != nil {
			t.Fatalf("Direct select error = %v", err)
		}
		t.Logf("Raw tags from DB: %q", raw.Tags)

		found, err := repo.FindByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if found.Amount != 1500 {
			t.Errorf("Amount = %v, want 1500", found.Amount)
		}

		t.Logf("Found tags: %v", found.Tags)
		if len(found.Tags) != 2 {
			t.Errorf("Tags count = %v, want 2 (tags: %v)", len(found.Tags), found.Tags)
		}
	})

	t.Run("正常系_タグなし", func(t *testing.T) {
		entry := entity.NewExpenseEntry(
			uuid.NewString(),
			time.Now(),
			"交通費",
			500,
			"バス代",
			[]string{},
		)

		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := repo.FindByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}

		if len(found.Tags) != 0 {
			t.Errorf("Tags count = %v, want 0", len(found.Tags))
		}
	})
}

func TestBunExpenseRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	// テストデータ作成
	for i := 0; i < 5; i++ {
		entry := entity.NewExpenseEntry(
			uuid.NewString(),
			time.Now().Add(time.Duration(i)*time.Hour),
			"食費",
			1000+i*100,
			"Test expense "+string(rune('A'+i)),
			[]string{},
		)
		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	tests := []struct {
		name      string
		limit     int
		offset    int
		wantCount int
	}{
		{
			name:      "正常系: 全件取得",
			limit:     10,
			offset:    0,
			wantCount: 5,
		},
		{
			name:      "正常系: limit=2",
			limit:     2,
			offset:    0,
			wantCount: 2,
		},
		{
			name:      "正常系: offset=3",
			limit:     10,
			offset:    3,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := repo.FindAll(ctx, tt.limit, tt.offset)
			if err != nil {
				t.Fatalf("FindAll() error = %v", err)
			}

			if len(entries) != tt.wantCount {
				t.Errorf("FindAll() count = %d, want %d", len(entries), tt.wantCount)
			}
		})
	}
}

func TestBunExpenseRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	entry := entity.NewExpenseEntry(
		uuid.NewString(),
		time.Now(),
		"食費",
		1500,
		"Test expense",
		[]string{"tag1", "tag2"},
	)

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.ID != entry.ID {
		t.Errorf("ID = %s, want %s", found.ID, entry.ID)
	}

	if found.Amount != entry.Amount {
		t.Errorf("Amount = %d, want %d", found.Amount, entry.Amount)
	}

	if len(found.Tags) != len(entry.Tags) {
		t.Errorf("Tags count = %d, want %d", len(found.Tags), len(entry.Tags))
	}
}

func TestBunExpenseRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	_, err = repo.FindByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent ID, got nil")
	}
}

func TestBunExpenseRepository_FindByDateRange(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	now := time.Now()
	// 過去、現在、未来のデータを作成
	dates := []time.Time{
		now.Add(-48 * time.Hour),
		now.Add(-24 * time.Hour),
		now,
		now.Add(24 * time.Hour),
		now.Add(48 * time.Hour),
	}

	for i, date := range dates {
		entry := entity.NewExpenseEntry(
			uuid.NewString(),
			date,
			"食費",
			1000+i*100,
			"",
			[]string{},
		)
		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	tests := []struct {
		name      string
		start     time.Time
		end       time.Time
		wantCount int
	}{
		{
			name:      "正常系: 過去1日",
			start:     now.Add(-25 * time.Hour),
			end:       now.Add(-23 * time.Hour),
			wantCount: 1,
		},
		{
			name:      "正常系: 現在±1日",
			start:     now.Add(-25 * time.Hour),
			end:       now.Add(25 * time.Hour),
			wantCount: 3,
		},
		{
			name:      "正常系: 全期間",
			start:     now.Add(-72 * time.Hour),
			end:       now.Add(72 * time.Hour),
			wantCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := repo.FindByDateRange(ctx, tt.start, tt.end)
			if err != nil {
				t.Fatalf("FindByDateRange() error = %v", err)
			}

			if len(entries) != tt.wantCount {
				t.Errorf("FindByDateRange() count = %d, want %d", len(entries), tt.wantCount)
			}
		})
	}
}

func TestBunExpenseRepository_FindByCategory(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	// テストデータ作成
	entries := []*entity.ExpenseEntry{
		entity.NewExpenseEntry(uuid.NewString(), time.Now(), "食費", 1000, "食品", []string{}),
		entity.NewExpenseEntry(uuid.NewString(), time.Now(), "食費", 2000, "外食", []string{}),
		entity.NewExpenseEntry(uuid.NewString(), time.Now(), "交通費", 500, "電車", []string{}),
	}

	for _, e := range entries {
		if err := repo.Create(ctx, e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// カテゴリ検索
	found, err := repo.FindByCategory(ctx, "食費")
	if err != nil {
		t.Fatalf("FindByCategory() error = %v", err)
	}

	if len(found) != 2 {
		t.Errorf("Found count = %v, want 2", len(found))
	}
}

func TestBunExpenseRepository_Update(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	entry := entity.NewExpenseEntry(
		uuid.NewString(),
		time.Now(),
		"食費",
		1000,
		"Original",
		[]string{},
	)

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 更新
	entry.Amount = 2000
	entry.Description = "Updated"
	entry.Tags = []string{"updated", "test"}

	if err := repo.Update(ctx, entry); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// 検証
	found, err := repo.FindByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Amount != 2000 {
		t.Errorf("Amount = %d, want 2000", found.Amount)
	}

	if found.Description != "Updated" {
		t.Errorf("Description = %s, want Updated", found.Description)
	}

	if len(found.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(found.Tags))
	}
}

func TestBunExpenseRepository_Delete(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunExpenseRepositoryWithDB(db)

	entry := entity.NewExpenseEntry(
		uuid.NewString(),
		time.Now(),
		"食費",
		1000,
		"",
		[]string{},
	)

	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 削除
	if err := repo.Delete(ctx, entry.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 検証
	_, err = repo.FindByID(ctx, entry.ID)
	if err == nil {
		t.Error("Expected error after delete, got nil")
	}
}

func TestBunExpenseRepository_Close(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	repo := NewBunExpenseRepositoryWithDB(db)

	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// =============================================================================
// BunCategoryRepository Tests
// =============================================================================

func TestNewBunCategoryRepository(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.MySQLConfig
		wantErr bool
	}{
		{
			name: "異常系: 無効なホスト",
			cfg: &config.MySQLConfig{
				Host:     "invalid-host-that-does-not-exist",
				Port:     3306,
				User:     "root",
				Password: "password",
				Database: "test",
			},
			wantErr: true,
		},
		{
			name: "異常系: 無効なポート",
			cfg: &config.MySQLConfig{
				Host:     "localhost",
				Port:     99999,
				User:     "root",
				Password: "password",
				Database: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewBunCategoryRepository(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBunCategoryRepository() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && repo == nil {
				t.Error("Expected non-nil repository")
			}

			if repo != nil {
				_ = repo.Close()
			}
		})
	}
}

func TestBunCategoryRepository_Create(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunCategoryRepositoryWithDB(db)

	category := entity.NewCategory(
		uuid.NewString(),
		"食費_"+uuid.NewString()[:8],
		"食品・飲料",
		"#FF0000",
	)

	if err := repo.Create(ctx, category); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 検証
	found, err := repo.FindByID(ctx, category.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Name != category.Name {
		t.Errorf("Name = %s, want %s", found.Name, category.Name)
	}

	if found.Description != category.Description {
		t.Errorf("Description = %s, want %s", found.Description, category.Description)
	}
}

func TestBunCategoryRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunCategoryRepositoryWithDB(db)

	category := entity.NewCategory(
		uuid.NewString(),
		"交通費_"+uuid.NewString()[:8],
		"",
		"",
	)

	if err := repo.Create(ctx, category); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(ctx, category.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.ID != category.ID {
		t.Errorf("ID = %s, want %s", found.ID, category.ID)
	}
}

func TestBunCategoryRepository_FindByID_NotFound(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunCategoryRepositoryWithDB(db)

	_, err = repo.FindByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent ID, got nil")
	}
}

func TestBunCategoryRepository_FindAll(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunCategoryRepositoryWithDB(db)

	// テストデータ作成
	categories := []string{"食費", "交通費", "娯楽費", "医療費", "教育費"}
	for _, name := range categories {
		uniqueName := name + "_" + uuid.NewString()[:8]
		category := entity.NewCategory(uuid.NewString(), uniqueName, "", "")
		if err := repo.Create(ctx, category); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// 全件取得
	found, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(found) < len(categories) {
		t.Errorf("FindAll() count = %d, want at least %d", len(found), len(categories))
	}
}

func TestBunCategoryRepository_FindByName(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewBunCategoryRepositoryWithDB(db)

	// デフォルトカテゴリが存在するはず
	category, err := repo.FindByName(ctx, "食費")
	if err != nil {
		t.Fatalf("FindByName() error = %v", err)
	}

	if category.Name != "食費" {
		t.Errorf("Name = %v, want 食費", category.Name)
	}
}

func TestBunCategoryRepository_Update(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunCategoryRepositoryWithDB(db)

	category := entity.NewCategory(
		uuid.NewString(),
		"食費_"+uuid.NewString()[:8],
		"Original",
		"",
	)

	if err := repo.Create(ctx, category); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 更新
	category.Description = "Updated"
	category.Color = "#00FF00"

	if err := repo.Update(ctx, category); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// 検証
	found, err := repo.FindByID(ctx, category.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if found.Description != "Updated" {
		t.Errorf("Description = %s, want Updated", found.Description)
	}

	if found.Color != "#00FF00" {
		t.Errorf("Color = %s, want #00FF00", found.Color)
	}
}

func TestBunCategoryRepository_Delete(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunCategoryRepositoryWithDB(db)

	category := entity.NewCategory(
		uuid.NewString(),
		"削除テスト",
		"",
		"",
	)

	if err := repo.Create(ctx, category); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 削除
	if err := repo.Delete(ctx, category.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 検証
	_, err = repo.FindByID(ctx, category.ID)
	if err == nil {
		t.Error("Expected error after delete, got nil")
	}
}

func TestBunCategoryRepository_Close(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	repo := NewBunCategoryRepositoryWithDB(db)

	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// =============================================================================
// Pagination Tests
// =============================================================================

func TestBunRepositories_Pagination(t *testing.T) {
	ctx := context.Background()
	db, err := NewTestDB(ctx)
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	defer func() { _ = db.Close() }()
	defer func() { _ = CleanupTestTables(ctx, db) }()

	repo := NewBunReceiptRepositoryWithDB(db)

	// 10件のテストデータ作成
	for i := 0; i < 10; i++ {
		receipt := entity.NewReceipt(
			uuid.NewString(),
			"ストア",
			time.Now(),
			1000*(i+1),
			100*(i+1),
			"食費",
		)
		if err := repo.Create(ctx, receipt); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// ページネーション（最初の5件）
	receipts, err := repo.FindAll(ctx, 5, 0)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(receipts) != 5 {
		t.Errorf("Found count = %v, want 5", len(receipts))
	}

	// 2ページ目
	receipts2, err := repo.FindAll(ctx, 5, 5)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}

	if len(receipts2) != 5 {
		t.Errorf("Found count = %v, want 5", len(receipts2))
	}
}

// =============================================================================
// Entity Validation Tests
// =============================================================================

func TestEntity_Validation(t *testing.T) {
	t.Run("Receipt_IsValid", func(t *testing.T) {
		tests := []struct {
			name      string
			storeName string
			amount    int
			want      bool
		}{
			{"正常", "ストア", 1000, true},
			{"空のストア名", "", 1000, false},
			{"負の金額", "ストア", -100, false},
			{"ゼロ金額", "ストア", 0, true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				receipt := entity.NewReceipt(
					uuid.NewString(),
					tt.storeName,
					time.Now(),
					tt.amount,
					0,
					"",
				)

				if got := receipt.IsValid(); got != tt.want {
					t.Errorf("IsValid() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("ReceiptItem_IsValid", func(t *testing.T) {
		tests := []struct {
			name     string
			itemName string
			quantity int
			price    int
			want     bool
		}{
			{"正常", "商品", 1, 100, true},
			{"空の商品名", "", 1, 100, false},
			{"ゼロ数量", "商品", 0, 100, false},
			{"負の価格", "商品", 1, -100, false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				item := entity.NewReceiptItem(
					uuid.NewString(),
					uuid.NewString(),
					tt.itemName,
					tt.quantity,
					tt.price,
				)

				if got := item.IsValid(); got != tt.want {
					t.Errorf("IsValid() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}
