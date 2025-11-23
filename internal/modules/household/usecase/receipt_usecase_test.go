package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"vision-api-app/internal/modules/household/domain/entity"
	"vision-api-app/internal/modules/vision/domain"
)

// MockAIRepository モックAIリポジトリ
type MockAIRepository struct {
	RecognizeReceiptFunc func(imageData []byte) (*domain.AIResult, error)
}

func (m *MockAIRepository) Correct(text string) (*domain.AIResult, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAIRepository) RecognizeImage(imageData []byte) (*domain.AIResult, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAIRepository) RecognizeReceipt(imageData []byte) (*domain.AIResult, error) {
	if m.RecognizeReceiptFunc != nil {
		return m.RecognizeReceiptFunc(imageData)
	}
	return domain.NewAIResult("", `{"store_name":"Test Store","purchase_date":"2025-11-23 12:00","total_amount":1000,"tax_amount":100,"items":[{"name":"Item1","quantity":1,"price":500}]}`, 10, 5, "test"), nil
}

func (m *MockAIRepository) CategorizeReceipt(receiptInfo string) (*domain.AIResult, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAIRepository) ProviderName() string {
	return "Mock AI Provider"
}

// MockReceiptRepository モックレシートリポジトリ
type MockReceiptRepository struct {
	CreateFunc   func(ctx context.Context, receipt *entity.Receipt) error
	FindByIDFunc func(ctx context.Context, id string) (*entity.Receipt, error)
	FindAllFunc  func(ctx context.Context, limit, offset int) ([]*entity.Receipt, error)
}

func (m *MockReceiptRepository) Create(ctx context.Context, receipt *entity.Receipt) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, receipt)
	}
	return nil
}

func (m *MockReceiptRepository) FindByID(ctx context.Context, id string) (*entity.Receipt, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return &entity.Receipt{ID: id}, nil
}

func (m *MockReceiptRepository) FindAll(ctx context.Context, limit, offset int) ([]*entity.Receipt, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc(ctx, limit, offset)
	}
	return []*entity.Receipt{}, nil
}

func (m *MockReceiptRepository) FindByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Receipt, error) {
	return nil, errors.New("not implemented")
}

func (m *MockReceiptRepository) Update(ctx context.Context, receipt *entity.Receipt) error {
	return errors.New("not implemented")
}

func (m *MockReceiptRepository) Delete(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

func TestNewReceiptUseCase(t *testing.T) {
	mockAI := &MockAIRepository{}
	mockReceipt := &MockReceiptRepository{}

	uc := NewReceiptUseCase(mockAI, mockReceipt)

	if uc == nil {
		t.Fatal("Expected non-nil usecase")
	}
	if uc.aiRepo == nil {
		t.Error("Expected aiRepo to be set")
	}
	if uc.receiptRepo == nil {
		t.Error("Expected receiptRepo to be set")
	}
}

func TestReceiptUseCase_ProcessReceiptImage(t *testing.T) {
	tests := []struct {
		name      string
		imageData []byte
		aiErr     error
		createErr error
		wantErr   bool
	}{
		{
			name:      "正常なレシート処理",
			imageData: []byte("image data"),
			aiErr:     nil,
			createErr: nil,
			wantErr:   false,
		},
		{
			name:      "AI認識エラー",
			imageData: []byte("image data"),
			aiErr:     errors.New("AI error"),
			createErr: nil,
			wantErr:   true,
		},
		{
			name:      "データベース保存エラー",
			imageData: []byte("image data"),
			aiErr:     nil,
			createErr: errors.New("DB error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAI := &MockAIRepository{
				RecognizeReceiptFunc: func(imageData []byte) (*domain.AIResult, error) {
					if tt.aiErr != nil {
						return nil, tt.aiErr
					}
					return domain.NewAIResult("", `{"store_name":"Test","purchase_date":"2025-11-23 12:00","total_amount":1000,"tax_amount":100,"items":[{"name":"Item","quantity":1,"price":1000}]}`, 10, 5, "test"), nil
				},
			}
			mockReceipt := &MockReceiptRepository{
				CreateFunc: func(ctx context.Context, receipt *entity.Receipt) error {
					return tt.createErr
				},
			}

			uc := NewReceiptUseCase(mockAI, mockReceipt)
			ctx := context.Background()

			receipt, err := uc.ProcessReceiptImage(ctx, tt.imageData)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessReceiptImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && receipt == nil {
				t.Error("Expected non-nil receipt")
			}
		})
	}
}

func TestReceiptUseCase_GetReceipt(t *testing.T) {
	mockAI := &MockAIRepository{}
	mockReceipt := &MockReceiptRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*entity.Receipt, error) {
			if id == "not-found" {
				return nil, errors.New("not found")
			}
			return &entity.Receipt{ID: id, StoreName: "Test Store"}, nil
		},
	}

	uc := NewReceiptUseCase(mockAI, mockReceipt)
	ctx := context.Background()

	// 正常ケース
	receipt, err := uc.GetReceipt(ctx, "test-id")
	if err != nil {
		t.Errorf("GetReceipt() error = %v", err)
	}
	if receipt == nil || receipt.ID != "test-id" {
		t.Error("Expected receipt with ID 'test-id'")
	}

	// エラーケース
	_, err = uc.GetReceipt(ctx, "not-found")
	if err == nil {
		t.Error("Expected error for not-found ID")
	}
}

func TestReceiptUseCase_ListReceipts(t *testing.T) {
	mockAI := &MockAIRepository{}
	mockReceipt := &MockReceiptRepository{
		FindAllFunc: func(ctx context.Context, limit, offset int) ([]*entity.Receipt, error) {
			return []*entity.Receipt{
				{ID: "1", StoreName: "Store1"},
				{ID: "2", StoreName: "Store2"},
			}, nil
		},
	}

	uc := NewReceiptUseCase(mockAI, mockReceipt)
	ctx := context.Background()

	receipts, err := uc.ListReceipts(ctx, 10, 0)
	if err != nil {
		t.Errorf("ListReceipts() error = %v", err)
	}
	if len(receipts) != 2 {
		t.Errorf("Expected 2 receipts, got %d", len(receipts))
	}
}

func TestReceiptUseCase_parseReceiptJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "正常なJSON",
			json:    `{"store_name":"Test","purchase_date":"2025-11-23 12:00","total_amount":1000,"tax_amount":100,"items":[{"name":"Item","quantity":1,"price":1000}]}`,
			wantErr: false,
		},
		{
			name:    "コードブロック付きJSON",
			json:    "```json\n{\"store_name\":\"Test\",\"purchase_date\":\"2025-11-23 12:00\",\"total_amount\":1000,\"tax_amount\":100,\"items\":[{\"name\":\"Item\",\"quantity\":1,\"price\":1000}]}\n```",
			wantErr: false,
		},
		{
			name:    "不正なJSON",
			json:    `{invalid json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAI := &MockAIRepository{}
			mockReceipt := &MockReceiptRepository{}
			uc := NewReceiptUseCase(mockAI, mockReceipt)

			receipt, err := uc.parseReceiptJSON(tt.json)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseReceiptJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && receipt == nil {
				t.Error("Expected non-nil receipt")
			}
		})
	}
}
