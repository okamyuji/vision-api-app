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
