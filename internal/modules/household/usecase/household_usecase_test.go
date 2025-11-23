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
			name: "正常なカテゴリ集計",
			receipts: []*entity.Receipt{
				{ID: "1", Category: "食費", TotalAmount: 1000},
				{ID: "2", Category: "食費", TotalAmount: 2000},
			},
			expenses: []*entity.ExpenseEntry{
				{ID: "3", Category: "交通費", Amount: 500},
			},
			receiptErr:   nil,
			expenseErr:   nil,
			wantErr:      false,
			wantCount:    2,
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
							if s.Total != 3000 { // 1000 + 2000
								t.Errorf("Expected total 3000 for category %s, got %d", tt.wantCategory, s.Total)
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
