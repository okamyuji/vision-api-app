package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"vision-api-app/internal/modules/household/domain/entity"
	"vision-api-app/internal/modules/shared/infrastructure/testcontainer"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
)

func setupTestDB(t *testing.T) (*bun.DB, func()) {
	t.Helper()
	ctx := context.Background()

	// TestContainer起動
	mysqlContainer, err := testcontainer.StartMySQL(ctx, t)
	if err != nil {
		t.Fatalf("Failed to start mysql container: %v", err)
	}

	// DB接続
	dsn := mysqlContainer.ConnectionString()
	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		_ = mysqlContainer.Close(ctx)
		t.Fatalf("Failed to open database: %v", err)
	}
	db := bun.NewDB(sqldb, mysqldialect.New())

	// テーブル作成
	if _, err := db.NewCreateTable().Model((*Receipt)(nil)).IfNotExists().Exec(ctx); err != nil {
		_ = mysqlContainer.Close(ctx)
		t.Fatalf("Failed to create receipts table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*ReceiptItem)(nil)).IfNotExists().Exec(ctx); err != nil {
		_ = mysqlContainer.Close(ctx)
		t.Fatalf("Failed to create receipt_items table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*ExpenseEntry)(nil)).IfNotExists().Exec(ctx); err != nil {
		_ = mysqlContainer.Close(ctx)
		t.Fatalf("Failed to create expense_entries table: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*Category)(nil)).IfNotExists().Exec(ctx); err != nil {
		_ = mysqlContainer.Close(ctx)
		t.Fatalf("Failed to create categories table: %v", err)
	}

	return db, func() {
		_ = db.Close()
		_ = mysqlContainer.Close(ctx)
	}
}

func TestBunReceiptRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunReceiptRepositoryWithDB(db)
	ctx := context.Background()

	now := time.Now()
	receipt := &entity.Receipt{
		ID:            "test-receipt-1",
		StoreName:     "テストストア",
		PurchaseDate:  now,
		TotalAmount:   1000,
		TaxAmount:     100,
		PaymentMethod: "現金",
		Category:      "食費",
		Items: []entity.ReceiptItem{
			{
				Name:     "商品A",
				Quantity: 2,
				Price:    500,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 保存
	err := repo.Create(ctx, receipt)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 取得して確認
	saved, err := repo.FindByID(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if saved.ID != receipt.ID {
		t.Errorf("ID = %v, want %v", saved.ID, receipt.ID)
	}
	if saved.StoreName != receipt.StoreName {
		t.Errorf("StoreName = %v, want %v", saved.StoreName, receipt.StoreName)
	}
	if saved.TotalAmount != receipt.TotalAmount {
		t.Errorf("TotalAmount = %v, want %v", saved.TotalAmount, receipt.TotalAmount)
	}
}

func TestBunReceiptRepository_FindByID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunReceiptRepositoryWithDB(db)
	ctx := context.Background()

	// テストデータ作成
	now := time.Now()
	receipt := &entity.Receipt{
		ID:           "test-find-1",
		StoreName:    "テストストア",
		PurchaseDate: now,
		TotalAmount:  1000,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Create(ctx, receipt); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "正常系: 存在するID",
			id:      "test-find-1",
			wantErr: false,
		},
		{
			name:    "異常系: 存在しないID",
			id:      "nonexistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.FindByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindByID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBunReceiptRepository_FindByDateRange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunReceiptRepositoryWithDB(db)
	ctx := context.Background()

	// テストデータ作成
	baseTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	receipts := []*entity.Receipt{
		{
			ID:           "test-range-1",
			StoreName:    "ストア1",
			PurchaseDate: baseTime.AddDate(0, 0, -5),
			TotalAmount:  1000,
			CreatedAt:    baseTime,
			UpdatedAt:    baseTime,
		},
		{
			ID:           "test-range-2",
			StoreName:    "ストア2",
			PurchaseDate: baseTime,
			TotalAmount:  2000,
			CreatedAt:    baseTime,
			UpdatedAt:    baseTime,
		},
		{
			ID:           "test-range-3",
			StoreName:    "ストア3",
			PurchaseDate: baseTime.AddDate(0, 0, 5),
			TotalAmount:  3000,
			CreatedAt:    baseTime,
			UpdatedAt:    baseTime,
		},
	}

	for _, r := range receipts {
		if err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// 範囲検索
	start := baseTime.AddDate(0, 0, -10)
	end := baseTime.AddDate(0, 0, 10)
	found, err := repo.FindByDateRange(ctx, start, end)
	if err != nil {
		t.Fatalf("FindByDateRange() error = %v", err)
	}

	if len(found) != 3 {
		t.Errorf("Found %d receipts, want 3", len(found))
	}
}

func TestBunExpenseRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunExpenseRepositoryWithDB(db)
	ctx := context.Background()

	now := time.Now()
	entry := &entity.ExpenseEntry{
		ID:          "test-expense-1",
		Date:        now,
		Category:    "食費",
		Amount:      1500,
		Description: "ランチ",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 保存
	err := repo.Create(ctx, entry)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 取得して確認
	saved, err := repo.FindByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if saved.ID != entry.ID {
		t.Errorf("ID = %v, want %v", saved.ID, entry.ID)
	}
	if saved.Category != entry.Category {
		t.Errorf("Category = %v, want %v", saved.Category, entry.Category)
	}
	if saved.Amount != entry.Amount {
		t.Errorf("Amount = %v, want %v", saved.Amount, entry.Amount)
	}
}

func TestBunCategoryRepository_Create(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunCategoryRepositoryWithDB(db)
	ctx := context.Background()

	now := time.Now()
	category := &entity.Category{
		ID:          "test-category-1",
		Name:        "食費",
		Description: "食品・飲料",
		CreatedAt:   now,
	}

	// 保存
	err := repo.Create(ctx, category)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 取得して確認
	saved, err := repo.FindByID(ctx, category.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	if saved.ID != category.ID {
		t.Errorf("ID = %v, want %v", saved.ID, category.ID)
	}
	if saved.Name != category.Name {
		t.Errorf("Name = %v, want %v", saved.Name, category.Name)
	}
}

// TestBunReceiptRepository_Update レシートの更新テスト
func TestBunReceiptRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunReceiptRepositoryWithDB(db)
	ctx := context.Background()

	// 初期データ作成
	receipt := &entity.Receipt{
		ID:           "update-receipt-1",
		StoreName:    "Old Store",
		PurchaseDate: time.Now().Truncate(time.Second),
		TotalAmount:  1000,
		Items:        []entity.ReceiptItem{},
	}
	if err := repo.Create(ctx, receipt); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 更新
	receipt.StoreName = "New Store"
	receipt.TotalAmount = 2000
	if err := repo.Update(ctx, receipt); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// 確認
	updated, err := repo.FindByID(ctx, receipt.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.StoreName != "New Store" {
		t.Errorf("StoreName = %v, want %v", updated.StoreName, "New Store")
	}
	if updated.TotalAmount != 2000 {
		t.Errorf("TotalAmount = %v, want %v", updated.TotalAmount, 2000)
	}
}

// TestBunReceiptRepository_Delete レシートの削除テスト
func TestBunReceiptRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunReceiptRepositoryWithDB(db)
	ctx := context.Background()

	// 初期データ作成
	receipt := &entity.Receipt{
		ID:           "delete-receipt-1",
		StoreName:    "Test Store",
		PurchaseDate: time.Now().Truncate(time.Second),
		TotalAmount:  1000,
		Items:        []entity.ReceiptItem{},
	}
	if err := repo.Create(ctx, receipt); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 削除
	if err := repo.Delete(ctx, receipt.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 削除確認
	_, err := repo.FindByID(ctx, receipt.ID)
	if err == nil {
		t.Error("Expected error for deleted receipt")
	}
}

// TestBunExpenseRepository_FindAll 経費エントリの全件取得テスト
func TestBunExpenseRepository_FindAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunExpenseRepositoryWithDB(db)
	ctx := context.Background()

	// テストデータ作成
	for i := 0; i < 5; i++ {
		entry := &entity.ExpenseEntry{
			ID:          string(rune('a' + i)),
			Description: "Expense " + string(rune('A'+i)),
			Amount:      100 * (i + 1),
			Date:        time.Now().Add(time.Duration(i) * time.Hour).Truncate(time.Second),
			Category:    "Test",
		}
		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// 全件取得
	entries, err := repo.FindAll(ctx, 10, 0)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("FindAll() got %d entries, want 5", len(entries))
	}

	// ページネーション
	entries, err = repo.FindAll(ctx, 2, 0)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("FindAll() with limit got %d entries, want 2", len(entries))
	}
}

// TestBunExpenseRepository_FindByDateRange 経費エントリの日付範囲検索テスト
func TestBunExpenseRepository_FindByDateRange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunExpenseRepositoryWithDB(db)
	ctx := context.Background()

	now := time.Now().Truncate(time.Second)
	// テストデータ作成
	entries := []*entity.ExpenseEntry{
		{ID: "e1", Description: "E1", Amount: 100, Date: now.Add(-48 * time.Hour), Category: "Test"},
		{ID: "e2", Description: "E2", Amount: 200, Date: now.Add(-24 * time.Hour), Category: "Test"},
		{ID: "e3", Description: "E3", Amount: 300, Date: now, Category: "Test"},
		{ID: "e4", Description: "E4", Amount: 400, Date: now.Add(24 * time.Hour), Category: "Test"},
	}
	for _, entry := range entries {
		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// 範囲検索
	start := now.Add(-36 * time.Hour)
	end := now.Add(12 * time.Hour)
	found, err := repo.FindByDateRange(ctx, start, end)
	if err != nil {
		t.Fatalf("FindByDateRange() error = %v", err)
	}
	if len(found) != 2 {
		t.Errorf("FindByDateRange() got %d entries, want 2", len(found))
	}
}

// TestBunExpenseRepository_FindByCategory 経費エントリのカテゴリ検索テスト
func TestBunExpenseRepository_FindByCategory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunExpenseRepositoryWithDB(db)
	ctx := context.Background()

	// テストデータ作成
	entries := []*entity.ExpenseEntry{
		{ID: "e1", Description: "E1", Amount: 100, Date: time.Now(), Category: "Food"},
		{ID: "e2", Description: "E2", Amount: 200, Date: time.Now(), Category: "Food"},
		{ID: "e3", Description: "E3", Amount: 300, Date: time.Now(), Category: "Transport"},
	}
	for _, entry := range entries {
		if err := repo.Create(ctx, entry); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// カテゴリ検索
	found, err := repo.FindByCategory(ctx, "Food")
	if err != nil {
		t.Fatalf("FindByCategory() error = %v", err)
	}
	if len(found) != 2 {
		t.Errorf("FindByCategory() got %d entries, want 2", len(found))
	}
}

// TestBunExpenseRepository_Update 経費エントリの更新テスト
func TestBunExpenseRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunExpenseRepositoryWithDB(db)
	ctx := context.Background()

	// 初期データ作成
	entry := &entity.ExpenseEntry{
		ID:          "update-expense-1",
		Description: "Old Description",
		Amount:      1000,
		Date:        time.Now().Truncate(time.Second),
		Category:    "Old",
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 更新
	entry.Description = "New Description"
	entry.Amount = 2000
	entry.Category = "New"
	if err := repo.Update(ctx, entry); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// 確認
	updated, err := repo.FindByID(ctx, entry.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.Description != "New Description" {
		t.Errorf("Description = %v, want %v", updated.Description, "New Description")
	}
	if updated.Amount != 2000 {
		t.Errorf("Amount = %v, want %v", updated.Amount, 2000)
	}
}

// TestBunExpenseRepository_Delete 経費エントリの削除テスト
func TestBunExpenseRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunExpenseRepositoryWithDB(db)
	ctx := context.Background()

	// 初期データ作成
	entry := &entity.ExpenseEntry{
		ID:          "delete-expense-1",
		Description: "Test",
		Amount:      1000,
		Date:        time.Now().Truncate(time.Second),
		Category:    "Test",
	}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 削除
	if err := repo.Delete(ctx, entry.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 削除確認
	_, err := repo.FindByID(ctx, entry.ID)
	if err == nil {
		t.Error("Expected error for deleted entry")
	}
}

// TestBunCategoryRepository_FindAll カテゴリの全件取得テスト
func TestBunCategoryRepository_FindAll(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunCategoryRepositoryWithDB(db)
	ctx := context.Background()

	// テストデータ作成
	categories := []*entity.Category{
		{ID: "cat1", Name: "Food", Description: "Food items"},
		{ID: "cat2", Name: "Transport", Description: "Transportation"},
		{ID: "cat3", Name: "Entertainment", Description: "Entertainment"},
	}
	for _, cat := range categories {
		if err := repo.Create(ctx, cat); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// 全件取得
	found, err := repo.FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll() error = %v", err)
	}
	if len(found) != 3 {
		t.Errorf("FindAll() got %d categories, want 3", len(found))
	}
}

// TestBunCategoryRepository_FindByName カテゴリ名検索テスト
func TestBunCategoryRepository_FindByName(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunCategoryRepositoryWithDB(db)
	ctx := context.Background()

	// テストデータ作成
	category := &entity.Category{
		ID:          "cat1",
		Name:        "UniqueCategory",
		Description: "Unique",
	}
	if err := repo.Create(ctx, category); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 名前検索
	found, err := repo.FindByName(ctx, "UniqueCategory")
	if err != nil {
		t.Fatalf("FindByName() error = %v", err)
	}
	if found.ID != category.ID {
		t.Errorf("FindByName() got ID %v, want %v", found.ID, category.ID)
	}

	// 存在しない名前
	_, err = repo.FindByName(ctx, "NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent category")
	}
}

// TestBunCategoryRepository_Update カテゴリの更新テスト
func TestBunCategoryRepository_Update(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunCategoryRepositoryWithDB(db)
	ctx := context.Background()

	// 初期データ作成
	category := &entity.Category{
		ID:          "update-cat-1",
		Name:        "Old Name",
		Description: "Old Description",
	}
	if err := repo.Create(ctx, category); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 更新
	category.Name = "New Name"
	category.Description = "New Description"
	if err := repo.Update(ctx, category); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// 確認
	updated, err := repo.FindByID(ctx, category.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("Name = %v, want %v", updated.Name, "New Name")
	}
	if updated.Description != "New Description" {
		t.Errorf("Description = %v, want %v", updated.Description, "New Description")
	}
}

// TestBunCategoryRepository_Delete カテゴリの削除テスト
func TestBunCategoryRepository_Delete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunCategoryRepositoryWithDB(db)
	ctx := context.Background()

	// 初期データ作成
	category := &entity.Category{
		ID:          "delete-cat-1",
		Name:        "Delete Me",
		Description: "To be deleted",
	}
	if err := repo.Create(ctx, category); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// 削除
	if err := repo.Delete(ctx, category.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// 削除確認
	_, err := repo.FindByID(ctx, category.ID)
	if err == nil {
		t.Error("Expected error for deleted category")
	}
}

// TestBunReceiptRepository_Close Closeのテスト
func TestBunReceiptRepository_Close(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunReceiptRepositoryWithDB(db)
	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestBunExpenseRepository_Close Closeのテスト
func TestBunExpenseRepository_Close(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunExpenseRepositoryWithDB(db)
	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestBunCategoryRepository_Close Closeのテスト
func TestBunCategoryRepository_Close(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewBunCategoryRepositoryWithDB(db)
	if err := repo.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
