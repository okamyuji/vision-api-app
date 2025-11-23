package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"vision-api-app/internal/modules/household/domain/entity"
)

// MockExpenseRepository モック家計簿リポジトリ
type MockExpenseRepository struct {
	FindAllFunc func(ctx context.Context, limit, offset int) ([]*entity.ExpenseEntry, error)
}

func (m *MockExpenseRepository) Create(ctx context.Context, entry *entity.ExpenseEntry) error {
	return errors.New("not implemented")
}

func (m *MockExpenseRepository) FindByID(ctx context.Context, id string) (*entity.ExpenseEntry, error) {
	return nil, errors.New("not implemented")
}

func (m *MockExpenseRepository) FindAll(ctx context.Context, limit, offset int) ([]*entity.ExpenseEntry, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc(ctx, limit, offset)
	}
	return []*entity.ExpenseEntry{}, nil
}

func (m *MockExpenseRepository) FindByDateRange(ctx context.Context, start, end time.Time) ([]*entity.ExpenseEntry, error) {
	return nil, errors.New("not implemented")
}

func (m *MockExpenseRepository) FindByCategory(ctx context.Context, category string) ([]*entity.ExpenseEntry, error) {
	return nil, errors.New("not implemented")
}

func (m *MockExpenseRepository) Update(ctx context.Context, entry *entity.ExpenseEntry) error {
	return errors.New("not implemented")
}

func (m *MockExpenseRepository) Delete(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func TestNewHouseholdUseCase(t *testing.T) {
	mockReceipt := &MockReceiptRepository{}
	mockExpense := &MockExpenseRepository{}

	uc := NewHouseholdUseCase(mockReceipt, mockExpense)

	if uc == nil {
		t.Fatal("Expected non-nil usecase")
	}
	if uc.receiptRepo == nil {
		t.Error("Expected receiptRepo to be set")
	}
	if uc.expenseRepo == nil {
		t.Error("Expected expenseRepo to be set")
	}
}

func TestHouseholdUseCase_GetCategorySummary(t *testing.T) {
	tests := []struct {
		name         string
		receipts     []*entity.Receipt
		expenses     []*entity.ExpenseEntry
		receiptErr   error
		expenseErr   error
		wantErr      bool
		wantCount    int
		wantCategory string
	}{
		{
			name: "正常なカテゴリ集計（明細項目ベース）",
			receipts: []*entity.Receipt{
				{
					ID: "1",
					Items: []entity.ReceiptItem{
						{Name: "牛乳", Category: "食費", Price: 200, Quantity: 1},
						{Name: "パン", Category: "食費", Price: 150, Quantity: 2},
					},
				},
				{
					ID: "2",
					Items: []entity.ReceiptItem{
						{Name: "シャンプー", Category: "日用品", Price: 500, Quantity: 1},
					},
				},
			},
			expenses: []*entity.ExpenseEntry{
				{ID: "3", Category: "交通費", Amount: 500},
			},
			receiptErr:   nil,
			expenseErr:   nil,
			wantErr:      false,
			wantCount:    3, // 食費、日用品、交通費
			wantCategory: "食費",
		},
		{
			name:       "レシート取得エラー",
			receipts:   nil,
			expenses:   nil,
			receiptErr: errors.New("receipt error"),
			expenseErr: nil,
			wantErr:    true,
		},
		{
			name: "家計簿エントリ取得エラー",
			receipts: []*entity.Receipt{
				{ID: "1", Category: "食費", TotalAmount: 1000},
			},
			expenses:   nil,
			receiptErr: nil,
			expenseErr: errors.New("expense error"),
			wantErr:    true,
		},
		{
			name:       "空のデータ",
			receipts:   []*entity.Receipt{},
			expenses:   []*entity.ExpenseEntry{},
			receiptErr: nil,
			expenseErr: nil,
			wantErr:    false,
			wantCount:  0,
		},
		{
			name: "カテゴリー未設定の明細項目",
			receipts: []*entity.Receipt{
				{
					ID: "1",
					Items: []entity.ReceiptItem{
						{Name: "商品A", Category: "", Price: 100, Quantity: 1},
						{Name: "商品B", Category: "食費", Price: 200, Quantity: 1},
					},
				},
			},
			expenses:     []*entity.ExpenseEntry{},
			receiptErr:   nil,
			expenseErr:   nil,
			wantErr:      false,
			wantCount:    2, // 食費、その他
			wantCategory: "",
		},
		{
			name: "複数カテゴリーの混在",
			receipts: []*entity.Receipt{
				{
					ID: "1",
					Items: []entity.ReceiptItem{
						{Name: "牛乳", Category: "食費", Price: 200, Quantity: 1},
						{Name: "シャンプー", Category: "日用品", Price: 500, Quantity: 1},
						{Name: "風邪薬", Category: "医療費", Price: 1200, Quantity: 1},
					},
				},
			},
			expenses: []*entity.ExpenseEntry{
				{ID: "1", Category: "食費", Amount: 300},
				{ID: "2", Category: "交通費", Amount: 500},
			},
			receiptErr:   nil,
			expenseErr:   nil,
			wantErr:      false,
			wantCount:    4, // 食費、日用品、医療費、交通費
			wantCategory: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockReceipt := &MockReceiptRepository{
				FindAllFunc: func(ctx context.Context, limit, offset int) ([]*entity.Receipt, error) {
					if tt.receiptErr != nil {
						return nil, tt.receiptErr
					}
					return tt.receipts, nil
				},
			}
			mockExpense := &MockExpenseRepository{
				FindAllFunc: func(ctx context.Context, limit, offset int) ([]*entity.ExpenseEntry, error) {
					if tt.expenseErr != nil {
						return nil, tt.expenseErr
					}
					return tt.expenses, nil
				},
			}

			uc := NewHouseholdUseCase(mockReceipt, mockExpense)
			ctx := context.Background()

			summary, err := uc.GetCategorySummary(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetCategorySummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(summary) != tt.wantCount {
					t.Errorf("Expected %d categories, got %d", tt.wantCount, len(summary))
				}

				if tt.wantCount > 0 && tt.wantCategory != "" {
					found := false
					for _, s := range summary {
						if s.Category == tt.wantCategory {
							found = true
							// 明細項目ベースの集計: 牛乳(200*1) + パン(150*2) = 500
							if s.Total != 500 {
								t.Errorf("Expected total 500 for category %s, got %d", tt.wantCategory, s.Total)
							}
							break
						}
					}
					if !found {
						t.Errorf("Expected category %s not found", tt.wantCategory)
					}
				}
			}
		})
	}
}

// TestHouseholdUseCase_GetCategorySummary_LargeValues 大きな値でのオーバーフロー対策テスト
func TestHouseholdUseCase_GetCategorySummary_LargeValues(t *testing.T) {
	// 大きな値でもオーバーフローしないことを確認
	mockReceipt := &MockReceiptRepository{
		FindAllFunc: func(ctx context.Context, limit, offset int) ([]*entity.Receipt, error) {
			return []*entity.Receipt{
				{
					ID: "1",
					Items: []entity.ReceiptItem{
						// 大きな値の乗算でもオーバーフローしないことを確認
						{Name: "高額商品", Category: "食費", Price: 1000000, Quantity: 100},
					},
				},
			}, nil
		},
	}
	mockExpense := &MockExpenseRepository{
		FindAllFunc: func(ctx context.Context, limit, offset int) ([]*entity.ExpenseEntry, error) {
			return []*entity.ExpenseEntry{}, nil
		},
	}

	uc := NewHouseholdUseCase(mockReceipt, mockExpense)
	ctx := context.Background()

	summary, err := uc.GetCategorySummary(ctx)
	if err != nil {
		t.Fatalf("GetCategorySummary() error = %v", err)
	}

	if len(summary) != 1 {
		t.Errorf("Expected 1 category, got %d", len(summary))
	}

	if summary[0].Category != "食費" {
		t.Errorf("Expected category '食費', got '%s'", summary[0].Category)
	}

	expectedTotal := 100000000 // 1000000 * 100
	if summary[0].Total != expectedTotal {
		t.Errorf("Expected total %d, got %d", expectedTotal, summary[0].Total)
	}
}
