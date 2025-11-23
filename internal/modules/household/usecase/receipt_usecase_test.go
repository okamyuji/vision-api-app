package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"vision-api-app/internal/modules/household/domain/entity"
	"vision-api-app/internal/modules/vision/domain"
)

// MockAIRepository モックAIリポジトリ
type MockAIRepository struct {
	RecognizeReceiptFunc  func(imageData []byte) (*domain.AIResult, error)
	CategorizeReceiptFunc func(receiptInfo string) (*domain.AIResult, error)
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
	if m.CategorizeReceiptFunc != nil {
		return m.CategorizeReceiptFunc(receiptInfo)
	}
	return domain.NewAIResult("", `{"category":"その他"}`, 10, 5, "test"), nil
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

// MockCacheRepository モックキャッシュリポジトリ
type MockCacheRepository struct {
	GetFunc    func(ctx context.Context, key string) ([]byte, error)
	SetFunc    func(ctx context.Context, key string, value []byte, expiration time.Duration) error
	DeleteFunc func(ctx context.Context, key string) error
	ExistsFunc func(ctx context.Context, key string) (bool, error)
	CloseFunc  func() error
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return nil, errors.New("not found")
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, expiration)
	}
	return nil
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, key)
	}
	return nil
}

func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsFunc != nil {
		return m.ExistsFunc(ctx, key)
	}
	return false, nil
}

func (m *MockCacheRepository) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestNewReceiptUseCase(t *testing.T) {
	mockAI := &MockAIRepository{}
	mockReceipt := &MockReceiptRepository{}
	mockCache := &MockCacheRepository{}

	uc := NewReceiptUseCase(mockAI, mockReceipt, mockCache)

	if uc == nil {
		t.Fatal("Expected non-nil usecase")
	}
	if uc.aiRepo == nil {
		t.Error("Expected aiRepo to be set")
	}
	if uc.receiptRepo == nil {
		t.Error("Expected receiptRepo to be set")
	}
	if uc.cacheRepo == nil {
		t.Error("Expected cacheRepo to be set")
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
			imageData: []byte("unique image"),
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
				FindByIDFunc: func(ctx context.Context, id string) (*entity.Receipt, error) {
					// 既存のレシートは存在しないとする
					return nil, errors.New("not found")
				},
				CreateFunc: func(ctx context.Context, receipt *entity.Receipt) error {
					return tt.createErr
				},
			}
			mockCache := &MockCacheRepository{}

			uc := NewReceiptUseCase(mockAI, mockReceipt, mockCache)
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
	mockCache := &MockCacheRepository{}

	uc := NewReceiptUseCase(mockAI, mockReceipt, mockCache)
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
	mockCache := &MockCacheRepository{}

	uc := NewReceiptUseCase(mockAI, mockReceipt, mockCache)
	ctx := context.Background()

	receipts, err := uc.ListReceipts(ctx, 10, 0)
	if err != nil {
		t.Errorf("ListReceipts() error = %v", err)
	}
	if len(receipts) != 2 {
		t.Errorf("Expected 2 receipts, got %d", len(receipts))
	}
}

// TestReceiptUseCase_ProcessReceiptImage_Deduplication 重複排除のテスト
func TestReceiptUseCase_ProcessReceiptImage_Deduplication(t *testing.T) {
	mockAI := &MockAIRepository{
		RecognizeReceiptFunc: func(imageData []byte) (*domain.AIResult, error) {
			return domain.NewAIResult("", `{"store_name":"Test Store","purchase_date":"2025-11-23 12:00","total_amount":1000,"tax_amount":100,"items":[{"name":"Item1","quantity":1,"price":500},{"name":"Item2","quantity":2,"price":250}]}`, 10, 5, "test"), nil
		},
	}

	savedReceipts := make(map[string]*entity.Receipt)
	mockReceipt := &MockReceiptRepository{
		FindByIDFunc: func(ctx context.Context, id string) (*entity.Receipt, error) {
			if receipt, ok := savedReceipts[id]; ok {
				return receipt, nil
			}
			return nil, errors.New("not found")
		},
		CreateFunc: func(ctx context.Context, receipt *entity.Receipt) error {
			savedReceipts[receipt.ID] = receipt
			return nil
		},
	}
	mockCache := &MockCacheRepository{}

	uc := NewReceiptUseCase(mockAI, mockReceipt, mockCache)
	ctx := context.Background()

	imageData := []byte("test image data")

	// 1回目のアップロード
	receipt1, err := uc.ProcessReceiptImage(ctx, imageData)
	if err != nil {
		t.Fatalf("First ProcessReceiptImage() error = %v", err)
	}
	if receipt1 == nil {
		t.Fatal("First ProcessReceiptImage() returned nil")
	}

	// 2回目のアップロード（同じ画像）
	receipt2, err := uc.ProcessReceiptImage(ctx, imageData)
	if err != nil {
		t.Fatalf("Second ProcessReceiptImage() error = %v", err)
	}
	if receipt2 == nil {
		t.Fatal("Second ProcessReceiptImage() returned nil")
	}

	// 同じIDであることを確認
	if receipt1.ID != receipt2.ID {
		t.Errorf("Receipt IDs should be the same: got %s and %s", receipt1.ID, receipt2.ID)
	}

	// レシートが1件だけ保存されていることを確認
	if len(savedReceipts) != 1 {
		t.Errorf("Expected 1 receipt in storage, got %d", len(savedReceipts))
	}

	// レシートアイテムのIDが正しい形式であることを確認（45文字：36文字のレシートID + "-" + 8桁のインデックス）
	for _, item := range receipt1.Items {
		if len(item.ID) != 45 {
			t.Errorf("Item ID length should be 45, got %d: %s", len(item.ID), item.ID)
		}
		if item.ReceiptID != receipt1.ID {
			t.Errorf("Item ReceiptID should match receipt ID: got %s, want %s", item.ReceiptID, receipt1.ID)
		}
		// アイテムIDがレシートIDで始まることを確認
		if len(item.ID) >= len(receipt1.ID) && item.ID[:len(receipt1.ID)] != receipt1.ID {
			t.Errorf("Item ID should start with receipt ID: got %s, want prefix %s", item.ID, receipt1.ID)
		}
	}
}

// TestReceiptUseCase_generateDeterministicReceiptID 決定的なレシートID生成のテスト
func TestReceiptUseCase_generateDeterministicReceiptID(t *testing.T) {
	uc := NewReceiptUseCase(nil, nil, nil)

	tests := []struct {
		name      string
		imageData []byte
		wantLen   int
	}{
		{
			name:      "正常なID生成",
			imageData: []byte("test image"),
			wantLen:   36, // UUID形式の文字列長
		},
		{
			name:      "異なる画像で異なるID",
			imageData: []byte("different image"),
			wantLen:   36,
		},
		{
			name:      "空の画像データ",
			imageData: []byte(""),
			wantLen:   36,
		},
	}

	ids := make(map[string]bool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := uc.generateDeterministicReceiptID(tt.imageData)
			if len(id) != tt.wantLen {
				t.Errorf("generateDeterministicReceiptID() length = %d, want %d", len(id), tt.wantLen)
			}
			// UUID形式の文字列構造（8-4-4-4-12）を確認
			if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
				t.Errorf("generateDeterministicReceiptID() format invalid: %s", id)
			}
			// 16進数文字のみであることを確認（ハイフンを除く）
			for i, c := range id {
				if i == 8 || i == 13 || i == 18 || i == 23 {
					continue // ハイフンの位置はスキップ
				}
				if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
					t.Errorf("generateDeterministicReceiptID() contains non-hex character at position %d: %c", i, c)
				}
			}
			// 重複チェック
			if ids[id] {
				t.Errorf("generateDeterministicReceiptID() generated duplicate ID: %s", id)
			}
			ids[id] = true
		})
	}

	// 決定性のテスト：同じ画像データから常に同じIDが生成されることを確認
	t.Run("決定性の確認", func(t *testing.T) {
		imageData := []byte("same image")
		id1 := uc.generateDeterministicReceiptID(imageData)
		id2 := uc.generateDeterministicReceiptID(imageData)
		id3 := uc.generateDeterministicReceiptID(imageData)

		if id1 != id2 {
			t.Errorf("Same image should generate same ID: got %s and %s", id1, id2)
		}
		if id1 != id3 {
			t.Errorf("Same image should generate same ID: got %s and %s", id1, id3)
		}
	})

	// 異なる画像データから異なるIDが生成されることを確認
	t.Run("一意性の確認", func(t *testing.T) {
		id1 := uc.generateDeterministicReceiptID([]byte("image1"))
		id2 := uc.generateDeterministicReceiptID([]byte("image2"))
		id3 := uc.generateDeterministicReceiptID([]byte("image3"))

		if id1 == id2 || id1 == id3 || id2 == id3 {
			t.Errorf("Different images should generate different IDs: %s, %s, %s", id1, id2, id3)
		}
	})

	// 大きなデータでも正しく動作することを確認
	t.Run("大きなデータの処理", func(t *testing.T) {
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}
		id := uc.generateDeterministicReceiptID(largeData)
		if len(id) != 36 {
			t.Errorf("generateDeterministicReceiptID() with large data: length = %d, want 36", len(id))
		}
	})
}

// TestReceiptUseCase_categorizeReceiptItems 明細項目ごとのカテゴリー判定テスト
func TestReceiptUseCase_categorizeReceiptItems(t *testing.T) {
	tests := []struct {
		name           string
		receipt        *entity.Receipt
		aiResponse     string
		aiErr          error
		wantCategories []string
		wantErr        bool
	}{
		{
			name: "JSON配列形式",
			receipt: &entity.Receipt{
				StoreName: "スーパーマーケット",
				Items: []entity.ReceiptItem{
					{Name: "牛乳", Quantity: 1, Price: 200},
					{Name: "パン", Quantity: 2, Price: 150},
					{Name: "りんご", Quantity: 3, Price: 100},
				},
			},
			aiResponse:     `["食費", "食費", "食費"]`,
			aiErr:          nil,
			wantCategories: []string{"食費", "食費", "食費"},
			wantErr:        false,
		},
		{
			name: "JSONオブジェクト形式",
			receipt: &entity.Receipt{
				StoreName: "ドラッグストア",
				Items: []entity.ReceiptItem{
					{Name: "シャンプー", Quantity: 1, Price: 800},
					{Name: "風邪薬", Quantity: 1, Price: 1200},
					{Name: "お菓子", Quantity: 2, Price: 300},
				},
			},
			aiResponse:     `{"categories": ["日用品", "医療費", "食費"]}`,
			aiErr:          nil,
			wantCategories: []string{"日用品", "医療費", "食費"},
			wantErr:        false,
		},
		{
			name: "番号付きオブジェクト形式",
			receipt: &entity.Receipt{
				StoreName: "コンビニ",
				Items: []entity.ReceiptItem{
					{Name: "おにぎり", Quantity: 1, Price: 120},
					{Name: "コーヒー", Quantity: 1, Price: 150},
				},
			},
			aiResponse:     `{"1": "食費", "2": "食費"}`,
			aiErr:          nil,
			wantCategories: []string{"食費", "食費"},
			wantErr:        false,
		},
		{
			name: "プレーンテキスト形式",
			receipt: &entity.Receipt{
				StoreName: "書店",
				Items: []entity.ReceiptItem{
					{Name: "雑誌", Quantity: 1, Price: 500},
					{Name: "文房具", Quantity: 2, Price: 200},
				},
			},
			aiResponse:     "1. 娯楽費\n2. 日用品",
			aiErr:          nil,
			wantCategories: []string{"娯楽費", "日用品"},
			wantErr:        false,
		},
		{
			name: "コードブロック付きJSON",
			receipt: &entity.Receipt{
				StoreName: "家電量販店",
				Items: []entity.ReceiptItem{
					{Name: "USB ケーブル", Quantity: 1, Price: 800},
				},
			},
			aiResponse:     "```json\n[\"日用品\"]\n```",
			aiErr:          nil,
			wantCategories: []string{"日用品"},
			wantErr:        false,
		},
		{
			name: "AI APIエラー（デフォルトカテゴリーを設定）",
			receipt: &entity.Receipt{
				StoreName: "テスト店",
				Items: []entity.ReceiptItem{
					{Name: "商品A", Quantity: 1, Price: 100},
				},
			},
			aiResponse:     "",
			aiErr:          errors.New("AI error"),
			wantCategories: []string{"その他"}, // エラー時はデフォルトカテゴリー
			wantErr:        false,           // エラーハンドリングを変更したのでエラーにならない
		},
		{
			name: "パースエラー（デフォルトカテゴリーを設定）",
			receipt: &entity.Receipt{
				StoreName: "テスト店",
				Items: []entity.ReceiptItem{
					{Name: "商品A", Quantity: 1, Price: 100},
					{Name: "商品B", Quantity: 2, Price: 200},
				},
			},
			aiResponse:     "", // 空文字列でパースエラーを発生させる
			aiErr:          nil,
			wantCategories: []string{"その他", "その他"}, // パースエラー時はデフォルトカテゴリー
			wantErr:        false,                  // エラーハンドリングを変更したのでエラーにならない
		},
		{
			name: "空の明細",
			receipt: &entity.Receipt{
				StoreName: "テスト店",
				Items:     []entity.ReceiptItem{},
			},
			aiResponse:     "",
			aiErr:          nil,
			wantCategories: nil,
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAI := &MockAIRepository{}
			mockAI.CategorizeReceiptFunc = func(receiptInfo string) (*domain.AIResult, error) {
				if tt.aiErr != nil {
					return nil, tt.aiErr
				}
				return domain.NewAIResult("", tt.aiResponse, 10, 5, "test"), nil
			}

			uc := NewReceiptUseCase(mockAI, nil, nil)

			err := uc.categorizeReceiptItems(tt.receipt)

			if (err != nil) != tt.wantErr {
				t.Errorf("categorizeReceiptItems() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.wantCategories != nil {
				if len(tt.receipt.Items) != len(tt.wantCategories) {
					t.Errorf("Item count mismatch: got %d, want %d", len(tt.receipt.Items), len(tt.wantCategories))
					return
				}
				for i, item := range tt.receipt.Items {
					if item.Category != tt.wantCategories[i] {
						t.Errorf("Item[%d] category = %v, want %v", i, item.Category, tt.wantCategories[i])
					}
				}
			}
		})
	}
}

// TestReceiptUseCase_parseItemCategories カテゴリーパース機能のテスト
func TestReceiptUseCase_parseItemCategories(t *testing.T) {
	uc := NewReceiptUseCase(nil, nil, nil)

	tests := []struct {
		name           string
		response       string
		itemCount      int
		wantCategories []string
		wantErr        bool
	}{
		{
			name:           "JSON配列",
			response:       `["食費", "日用品", "医療費"]`,
			itemCount:      3,
			wantCategories: []string{"食費", "日用品", "医療費"},
			wantErr:        false,
		},
		{
			name:           "JSONオブジェクト",
			response:       `{"categories": ["食費", "日用品"]}`,
			itemCount:      2,
			wantCategories: []string{"食費", "日用品"},
			wantErr:        false,
		},
		{
			name:           "番号付きオブジェクト",
			response:       `{"1": "食費", "2": "日用品", "3": "医療費"}`,
			itemCount:      3,
			wantCategories: []string{"食費", "日用品", "医療費"},
			wantErr:        false,
		},
		{
			name:           "プレーンテキスト",
			response:       "1. 食費\n2. 日用品\n3. 医療費",
			itemCount:      3,
			wantCategories: []string{"食費", "日用品", "医療費"},
			wantErr:        false,
		},
		{
			name:           "コードブロック付き",
			response:       "```json\n[\"食費\", \"日用品\"]\n```",
			itemCount:      2,
			wantCategories: []string{"食費", "日用品"},
			wantErr:        false,
		},
		{
			name:           "オブジェクト配列形式",
			response:       `[{"item": "牛乳", "category": "食費"}, {"item": "シャンプー", "category": "日用品"}]`,
			itemCount:      2,
			wantCategories: []string{"食費", "日用品"},
			wantErr:        false,
		},
		{
			name:           "オブジェクト配列形式（詳細情報付き）",
			response:       `[{"item": "十六茶", "category": "食費", "confidence": 98, "reason": "飲料"}, {"item": "ベーコン", "category": "食費", "confidence": 95, "reason": "食品"}]`,
			itemCount:      2,
			wantCategories: []string{"食費", "食費"},
			wantErr:        false,
		},
		{
			name:           "不正な形式",
			response:       "invalid response",
			itemCount:      2,
			wantCategories: []string{"invalid response"},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			categories, err := uc.parseItemCategories(tt.response, tt.itemCount)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseItemCategories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(categories) != len(tt.wantCategories) {
					t.Errorf("parseItemCategories() length = %d, want %d", len(categories), len(tt.wantCategories))
					return
				}
				for i, cat := range categories {
					if cat != tt.wantCategories[i] {
						t.Errorf("parseItemCategories()[%d] = %v, want %v", i, cat, tt.wantCategories[i])
					}
				}
			}
		})
	}
}

// TestReceiptUseCase_ItemIDLength アイテムIDの長さが45文字であることを検証
func TestReceiptUseCase_ItemIDLength(t *testing.T) {
	mockAI := &MockAIRepository{}
	mockReceipt := &MockReceiptRepository{}
	mockCache := &MockCacheRepository{}
	uc := NewReceiptUseCase(mockAI, mockReceipt, mockCache)

	// 36文字のレシートIDを使用
	testReceiptID := "12345678-1234-1234-1234-123456789012"

	receiptJSON := `{
		"store_name": "Test Store",
		"purchase_date": "2025-11-23 12:00",
		"total_amount": 1000,
		"tax_amount": 100,
		"items": [
			{"name": "Item1", "quantity": 1, "price": 500},
			{"name": "Item2", "quantity": 2, "price": 250}
		]
	}`

	receipt, err := uc.parseReceiptJSON(receiptJSON, testReceiptID)
	if err != nil {
		t.Fatalf("parseReceiptJSON() error = %v", err)
	}

	if len(receipt.Items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(receipt.Items))
	}

	for i, item := range receipt.Items {
		// アイテムIDは45文字であることを確認
		if len(item.ID) != 45 {
			t.Errorf("Item[%d] ID length = %d, want 45: %s", i, len(item.ID), item.ID)
		}

		// アイテムIDがレシートIDで始まることを確認
		if item.ID[:36] != testReceiptID {
			t.Errorf("Item[%d] ID should start with receipt ID: got %s, want prefix %s", i, item.ID, testReceiptID)
		}

		// アイテムIDの形式を確認（36文字のレシートID + "-" + 8桁の数字）
		expectedID := fmt.Sprintf("%s-%08d", testReceiptID, i)
		if item.ID != expectedID {
			t.Errorf("Item[%d] ID = %s, want %s", i, item.ID, expectedID)
		}

		// データベース制約（VARCHAR(50)）に収まることを確認
		if len(item.ID) > 50 {
			t.Errorf("Item[%d] ID length %d exceeds database constraint VARCHAR(50)", i, len(item.ID))
		}
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
			mockCache := &MockCacheRepository{}
			uc := NewReceiptUseCase(mockAI, mockReceipt, mockCache)

			// UUID形式のレシートID（36文字）を使用
			testReceiptID := "12345678-1234-1234-1234-123456789012"
			receipt, err := uc.parseReceiptJSON(tt.json, testReceiptID)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseReceiptJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && receipt == nil {
				t.Error("Expected non-nil receipt")
			}

			// 正常ケースの場合、アイテムIDの長さを確認
			if !tt.wantErr && receipt != nil {
				for _, item := range receipt.Items {
					if len(item.ID) != 45 {
						t.Errorf("Item ID length should be 45, got %d: %s", len(item.ID), item.ID)
					}
					if item.ReceiptID != testReceiptID {
						t.Errorf("Item ReceiptID should match receipt ID: got %s, want %s", item.ReceiptID, testReceiptID)
					}
					// アイテムIDがレシートIDで始まることを確認
					if len(item.ID) >= len(testReceiptID) && item.ID[:len(testReceiptID)] != testReceiptID {
						t.Errorf("Item ID should start with receipt ID: got %s, want prefix %s", item.ID, testReceiptID)
					}
				}
			}
		})
	}
}
