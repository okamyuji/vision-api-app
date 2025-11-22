package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"vision-api-app/internal/domain/entity"
)

// MockAICorrectionUseCaseForVision Vision用のモック
type MockAICorrectionUseCaseForVision struct {
	RecognizeImageFunc    func([]byte) (*entity.AIResult, error)
	RecognizeReceiptFunc  func([]byte) (*entity.AIResult, error)
	CategorizeReceiptFunc func(string) (*entity.AIResult, error)
	GetProviderNameFunc   func() string
}

func (m *MockAICorrectionUseCaseForVision) Correct(text string) (*entity.AIResult, error) {
	return nil, errors.New("not implemented")
}

func (m *MockAICorrectionUseCaseForVision) RecognizeImage(imageData []byte) (*entity.AIResult, error) {
	if m.RecognizeImageFunc != nil {
		return m.RecognizeImageFunc(imageData)
	}
	return &entity.AIResult{
		OriginalText:  "",
		CorrectedText: "認識されたテキスト",
		InputTokens:   100,
		OutputTokens:  50,
	}, nil
}

func (m *MockAICorrectionUseCaseForVision) RecognizeReceipt(imageData []byte) (*entity.AIResult, error) {
	if m.RecognizeReceiptFunc != nil {
		return m.RecognizeReceiptFunc(imageData)
	}
	return &entity.AIResult{
		OriginalText:  "",
		CorrectedText: `{"store_name":"テストストア","total_amount":1500}`,
		InputTokens:   100,
		OutputTokens:  50,
	}, nil
}

func (m *MockAICorrectionUseCaseForVision) CategorizeReceipt(receiptInfo string) (*entity.AIResult, error) {
	if m.CategorizeReceiptFunc != nil {
		return m.CategorizeReceiptFunc(receiptInfo)
	}
	return &entity.AIResult{
		OriginalText:  receiptInfo,
		CorrectedText: `{"category":"食費","confidence":0.95}`,
		InputTokens:   50,
		OutputTokens:  30,
	}, nil
}

func (m *MockAICorrectionUseCaseForVision) GetProviderName() string {
	if m.GetProviderNameFunc != nil {
		return m.GetProviderNameFunc()
	}
	return "Claude"
}

// MockCacheRepository キャッシュリポジトリのモック
type MockCacheRepository struct {
	SetFunc    func(ctx context.Context, key string, value []byte, expiration time.Duration) error
	GetFunc    func(ctx context.Context, key string) ([]byte, error)
	DeleteFunc func(ctx context.Context, key string) error
	ExistsFunc func(ctx context.Context, key string) (bool, error)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, expiration)
	}
	return nil
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return nil, errors.New("not found")
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

// MockReceiptRepository レシートリポジトリのモック
type MockReceiptRepository struct {
	CreateFunc   func(ctx context.Context, receipt *entity.Receipt) error
	FindByIDFunc func(ctx context.Context, id string) (*entity.Receipt, error)
}

func (m *MockReceiptRepository) Create(ctx context.Context, receipt *entity.Receipt) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, receipt)
	}
	return nil
}

// TestVisionHandler_SaveReceiptToDatabase_TotalAmountCorrection total_amount補正のテスト
func TestVisionHandler_SaveReceiptToDatabase_TotalAmountCorrection(t *testing.T) {
	tests := []struct {
		name                string
		receiptJSON         string
		expectedTotalAmount int
		wantSave            bool
	}{
		{
			name: "正常系: total_amountがitemsの合計と一致",
			receiptJSON: `{
				"store_name": "テストストア",
				"purchase_date": "2025-11-22 14:30",
				"total_amount": 1500,
				"tax_amount": 150,
				"items": [
					{"name": "商品A", "quantity": 1, "price": 500},
					{"name": "商品B", "quantity": 2, "price": 500}
				]
			}`,
			expectedTotalAmount: 1500,
			wantSave:            true,
		},
		{
			name: "補正: total_amountがitemsの合計と不一致（Claudeの計算ミス）",
			receiptJSON: `{
				"store_name": "テストストア",
				"purchase_date": "2025-11-22 14:30",
				"total_amount": 2000,
				"tax_amount": 150,
				"items": [
					{"name": "商品A", "quantity": 1, "price": 500},
					{"name": "商品B", "quantity": 2, "price": 500}
				]
			}`,
			expectedTotalAmount: 1500,
			wantSave:            true,
		},
		{
			name: "補正: 「お預かり」が誤ってtotal_amountになっている",
			receiptJSON: `{
				"store_name": "ローソン",
				"purchase_date": "2025-11-22 12:28",
				"total_amount": 1000,
				"tax_amount": 0,
				"items": [
					{"name": "チョコ", "quantity": 1, "price": 130},
					{"name": "弁当", "quantity": 1, "price": 529},
					{"name": "おにぎり", "quantity": 2, "price": 341}
				]
			}`,
			expectedTotalAmount: 1341,
			wantSave:            true,
		},
		{
			name: "境界値: items が空",
			receiptJSON: `{
				"store_name": "テストストア",
				"purchase_date": "2025-11-22 14:30",
				"total_amount": 1500,
				"tax_amount": 150,
				"items": []
			}`,
			expectedTotalAmount: 1500,
			wantSave:            true,
		},
		{
			name: "正常系: 複数商品で数量が異なる",
			receiptJSON: `{
				"store_name": "スーパー",
				"purchase_date": "2025-11-22 14:30",
				"total_amount": 3000,
				"tax_amount": 0,
				"items": [
					{"name": "商品A", "quantity": 3, "price": 200},
					{"name": "商品B", "quantity": 1, "price": 800},
					{"name": "商品C", "quantity": 2, "price": 500}
				]
			}`,
			expectedTotalAmount: 2400,
			wantSave:            true,
		},
		{
			name: "異常系: 無効なJSON",
			receiptJSON: `{
				"store_name": "テストストア",
				invalid json
			}`,
			expectedTotalAmount: 0,
			wantSave:            false,
		},
		{
			name: "正常系: ```json```で囲まれている",
			receiptJSON: "```json\n" + `{
				"store_name": "テストストア",
				"purchase_date": "2025-11-22 14:30",
				"total_amount": 2000,
				"tax_amount": 150,
				"items": [
					{"name": "商品A", "quantity": 1, "price": 500},
					{"name": "商品B", "quantity": 2, "price": 500}
				]
			}` + "\n```",
			expectedTotalAmount: 1500,
			wantSave:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			savedReceipt := &entity.Receipt{}
			saveCalled := false

			mockRepo := &MockReceiptRepository{
				CreateFunc: func(ctx context.Context, receipt *entity.Receipt) error {
					saveCalled = true
					savedReceipt = receipt
					return nil
				},
			}

			handler := &VisionHandler{
				receiptRepo: mockRepo,
			}

			// saveReceiptToDatabaseを実行
			ctx := context.Background()
			handler.saveReceiptToDatabase(ctx, tt.receiptJSON)

			// 少し待機（goroutineの完了を待つ）
			time.Sleep(100 * time.Millisecond)

			if tt.wantSave {
				if !saveCalled {
					t.Error("Expected Create to be called, but it wasn't")
					return
				}

				if savedReceipt.TotalAmount != tt.expectedTotalAmount {
					t.Errorf("TotalAmount = %d, want %d", savedReceipt.TotalAmount, tt.expectedTotalAmount)
				}

				// 商品の合計とtotal_amountが一致することを確認
				if len(savedReceipt.Items) > 0 {
					itemsTotal := 0
					for _, item := range savedReceipt.Items {
						itemsTotal += item.Price * item.Quantity
					}
					if savedReceipt.TotalAmount != itemsTotal {
						t.Errorf("TotalAmount (%d) does not match items total (%d)", savedReceipt.TotalAmount, itemsTotal)
					}
				}
			} else {
				if saveCalled {
					t.Error("Expected Create not to be called, but it was")
				}
			}
		})
	}
}

// TestVisionHandler_SaveReceiptToDatabase_DateParsing 日付パースのテスト
func TestVisionHandler_SaveReceiptToDatabase_DateParsing(t *testing.T) {
	tests := []struct {
		name         string
		purchaseDate string
		wantError    bool
	}{
		{
			name:         "正常系: YYYY-MM-DD HH:MM形式",
			purchaseDate: "2025-11-22 14:30",
			wantError:    false,
		},
		{
			name:         "正常系: YYYY-MM-DD形式",
			purchaseDate: "2025-11-22",
			wantError:    false,
		},
		{
			name:         "正常系: 空文字（デフォルト値）",
			purchaseDate: "",
			wantError:    false,
		},
		{
			name:         "異常系: 無効な日付形式",
			purchaseDate: "invalid-date",
			wantError:    false, // デフォルト値が使用される
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			savedReceipt := &entity.Receipt{}

			mockRepo := &MockReceiptRepository{
				CreateFunc: func(ctx context.Context, receipt *entity.Receipt) error {
					savedReceipt = receipt
					return nil
				},
			}

			handler := &VisionHandler{
				receiptRepo: mockRepo,
			}

			receiptJSON := `{
				"store_name": "テストストア",
				"purchase_date": "` + tt.purchaseDate + `",
				"total_amount": 1000,
				"items": [{"name": "商品", "quantity": 1, "price": 1000}]
			}`

			ctx := context.Background()
			handler.saveReceiptToDatabase(ctx, receiptJSON)

			time.Sleep(100 * time.Millisecond)

			if savedReceipt.StoreName == "" {
				if !tt.wantError {
					t.Error("Expected receipt to be saved, but it wasn't")
				}
				return
			}

			if savedReceipt.PurchaseDate.IsZero() {
				t.Error("PurchaseDate should not be zero")
			}
		})
	}
}

// TestVisionHandler_SaveReceiptToDatabase_ErrorHandling エラーハンドリングのテスト
func TestVisionHandler_SaveReceiptToDatabase_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		receiptJSON string
		repoError   error
		wantPanic   bool
	}{
		{
			name: "正常系: リポジトリエラーでもpanicしない",
			receiptJSON: `{
				"store_name": "テストストア",
				"purchase_date": "2025-11-22 14:30",
				"total_amount": 1000,
				"items": [{"name": "商品", "quantity": 1, "price": 1000}]
			}`,
			repoError: errors.New("database error"),
			wantPanic: false,
		},
		{
			name:        "異常系: 空のJSON",
			receiptJSON: "",
			repoError:   nil,
			wantPanic:   false,
		},
		{
			name:        "異常系: null",
			receiptJSON: "null",
			repoError:   nil,
			wantPanic:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantPanic {
						t.Errorf("Unexpected panic: %v", r)
					}
				}
			}()

			mockRepo := &MockReceiptRepository{
				CreateFunc: func(ctx context.Context, receipt *entity.Receipt) error {
					return tt.repoError
				},
			}

			handler := &VisionHandler{
				receiptRepo: mockRepo,
			}

			ctx := context.Background()
			handler.saveReceiptToDatabase(ctx, tt.receiptJSON)

			time.Sleep(100 * time.Millisecond)
		})
	}
}

func (m *MockReceiptRepository) FindByID(ctx context.Context, id string) (*entity.Receipt, error) {
	if m.FindByIDFunc != nil {
		return m.FindByIDFunc(ctx, id)
	}
	return nil, errors.New("not found")
}

// TestVisionHandler_WithCache Redisキャッシュ機能のテスト
func TestVisionHandler_WithCache(t *testing.T) {
	t.Run("キャッシュヒット時", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		mockCache := &MockCacheRepository{
			GetFunc: func(ctx context.Context, key string) ([]byte, error) {
				return []byte("cached result"), nil
			},
		}

		handler := NewVisionHandler(mockAI, mockCache, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		_, _ = part.Write([]byte("test image data"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		if rec.Header().Get("X-Cache") != "HIT" {
			t.Error("Expected X-Cache: HIT header")
		}
	})

	t.Run("キャッシュミス時", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{
			RecognizeImageFunc: func(data []byte) (*entity.AIResult, error) {
				return &entity.AIResult{
					CorrectedText: "new result",
					InputTokens:   100,
					OutputTokens:  50,
				}, nil
			},
		}
		mockCache := &MockCacheRepository{
			GetFunc: func(ctx context.Context, key string) ([]byte, error) {
				return nil, errors.New("not found")
			},
			SetFunc: func(ctx context.Context, key string, value []byte, expiration time.Duration) error {
				return nil
			},
		}

		handler := NewVisionHandler(mockAI, mockCache, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		_, _ = part.Write([]byte("test image data"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		if rec.Header().Get("X-Cache") != "MISS" {
			t.Error("Expected X-Cache: MISS header")
		}
	})
}

// TestVisionHandler_WithReceiptSave レシート保存機能のテスト
func TestVisionHandler_WithReceiptSave(t *testing.T) {
	t.Run("レシート保存成功", func(t *testing.T) {
		saved := false
		mockAI := &MockAICorrectionUseCaseForVision{
			RecognizeReceiptFunc: func(data []byte) (*entity.AIResult, error) {
				return &entity.AIResult{
					CorrectedText: `{"store_name":"TestStore","purchase_date":"2025-11-22 10:00","total_amount":1000,"tax_amount":100,"items":[{"name":"item1","quantity":1,"price":500}]}`,
					InputTokens:   100,
					OutputTokens:  50,
				}, nil
			},
		}
		mockReceipt := &MockReceiptRepository{
			CreateFunc: func(ctx context.Context, receipt *entity.Receipt) error {
				saved = true
				return nil
			},
		}

		handler := NewVisionHandler(mockAI, nil, mockReceipt)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		_, _ = part.Write([]byte("test image data"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/receipt", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleReceiptAnalyze(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		// バックグラウンド処理の完了を待つ
		time.Sleep(100 * time.Millisecond)

		if !saved {
			t.Error("Expected receipt to be saved")
		}
	})
}

// TestGenerateCacheKey キャッシュキー生成のテスト
func TestGenerateCacheKey(t *testing.T) {
	mockAI := &MockAICorrectionUseCaseForVision{}
	handler := NewVisionHandler(mockAI, nil, nil)

	key1 := handler.generateCacheKey("test", []byte("data1"))
	key2 := handler.generateCacheKey("test", []byte("data2"))
	key3 := handler.generateCacheKey("test", []byte("data1"))

	if key1 == key2 {
		t.Error("Different data should generate different keys")
	}

	if key1 != key3 {
		t.Error("Same data should generate same keys")
	}

	if key1[:7] != "vision:" {
		t.Error("Key should start with 'vision:'")
	}
}

// TestGenerateUUID UUID生成のテスト
func TestGenerateUUID(t *testing.T) {
	uuid1 := generateUUID()
	uuid2 := generateUUID()

	if uuid1 == uuid2 {
		t.Error("UUIDs should be unique")
	}

	if len(uuid1) != 36 {
		t.Errorf("UUID length should be 36, got %d", len(uuid1))
	}
}

func TestVisionHandler_HandleAnalyze(t *testing.T) {
	t.Run("正常系_画像解析成功", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{
			RecognizeImageFunc: func(data []byte) (*entity.AIResult, error) {
				return &entity.AIResult{
					OriginalText:  "",
					CorrectedText: "テスト結果",
					InputTokens:   150,
					OutputTokens:  75,
				}, nil
			},
		}

		handler := NewVisionHandler(mockAI, nil, nil)

		// マルチパートフォームの作成
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		_, _ = part.Write([]byte("fake image data"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
		}

		var response VisionResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Success should be true")
		}
		if response.Text != "テスト結果" {
			t.Errorf("Text = %v, want テスト結果", response.Text)
		}
		if response.Tokens.InputTokens != 150 {
			t.Errorf("InputTokens = %v, want 150", response.Tokens.InputTokens)
		}
	})

	t.Run("異常系_GETメソッド", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/vision/analyze", nil)
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("異常系_画像ファイルなし", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("異常系_Vision APIエラー", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{
			RecognizeImageFunc: func(data []byte) (*entity.AIResult, error) {
				return nil, errors.New("API error")
			},
		}

		handler := NewVisionHandler(mockAI, nil, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		_, _ = part.Write([]byte("fake image data"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusInternalServerError)
		}

		var response VisionResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Success {
			t.Error("Success should be false")
		}
		if response.Error == "" {
			t.Error("Error message should not be empty")
		}
	})

	t.Run("異常系_不正なフォーム", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=invalid")
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("境界値_大きな画像", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "large.png")
		// 5MBの画像データ
		largeData := make([]byte, 5*1024*1024)
		_, _ = part.Write(largeData)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
		}
	})

	t.Run("エッジケース_空の画像データ", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "empty.png")
		_, _ = part.Write([]byte(""))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleAnalyze(rec, req)

		// 空データでもAPIに渡される（API側でエラー判定）
		if rec.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
		}
	})
}

func TestVisionHandler_sendError(t *testing.T) {
	mockAI := &MockAICorrectionUseCaseForVision{}
	handler := NewVisionHandler(mockAI, nil, nil)

	tests := []struct {
		name       string
		message    string
		statusCode int
	}{
		{"400エラー", "Bad Request", http.StatusBadRequest},
		{"500エラー", "Internal Server Error", http.StatusInternalServerError},
		{"404エラー", "Not Found", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.sendError(rec, tt.message, tt.statusCode)

			if rec.Code != tt.statusCode {
				t.Errorf("Status = %v, want %v", rec.Code, tt.statusCode)
			}

			var response VisionResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Success {
				t.Error("Success should be false")
			}
			if response.Error != tt.message {
				t.Errorf("Error = %v, want %v", response.Error, tt.message)
			}
		})
	}
}

func TestVisionResponse_JSON(t *testing.T) {
	response := VisionResponse{
		Success: true,
		Text:    "テキスト",
		Tokens: &AITokensResponse{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded VisionResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.Success != response.Success {
		t.Errorf("Success = %v, want %v", decoded.Success, response.Success)
	}
	if decoded.Text != response.Text {
		t.Errorf("Text = %v, want %v", decoded.Text, response.Text)
	}
	if decoded.Tokens.TotalTokens != response.Tokens.TotalTokens {
		t.Errorf("TotalTokens = %v, want %v", decoded.Tokens.TotalTokens, response.Tokens.TotalTokens)
	}
}

func TestAITokensResponse(t *testing.T) {
	tokens := AITokensResponse{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	if tokens.InputTokens != 100 {
		t.Errorf("InputTokens = %v, want 100", tokens.InputTokens)
	}
	if tokens.OutputTokens != 50 {
		t.Errorf("OutputTokens = %v, want 50", tokens.OutputTokens)
	}
	if tokens.TotalTokens != 150 {
		t.Errorf("TotalTokens = %v, want 150", tokens.TotalTokens)
	}
}

func TestVisionHandler_ReadImageError(t *testing.T) {
	mockAI := &MockAICorrectionUseCaseForVision{}
	handler := NewVisionHandler(mockAI, nil, nil)

	// 読み込みエラーを発生させるためのカスタムReader
	errorReader := &errorReader{}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", "test.png")
	_, _ = part.Write([]byte("test"))
	_ = writer.Close()

	// 正常なリクエストを作成してから、Bodyを差し替え
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Bodyを読み込みエラーを起こすReaderに差し替え
	req.Body = io.NopCloser(errorReader)

	rec := httptest.NewRecorder()
	handler.HandleAnalyze(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", rec.Code, http.StatusBadRequest)
	}
}

// errorReader 読み込み時にエラーを返すReader
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

func TestVisionHandler_HandleReceiptAnalyze(t *testing.T) {
	t.Run("正常系_レシート認識成功", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{
			RecognizeReceiptFunc: func(data []byte) (*entity.AIResult, error) {
				return &entity.AIResult{
					OriginalText:  "",
					CorrectedText: `{"store_name":"スーパーマーケット","total_amount":1500,"items":[{"name":"野菜","quantity":1,"price":500}]}`,
					InputTokens:   150,
					OutputTokens:  100,
				}, nil
			},
		}

		handler := NewVisionHandler(mockAI, nil, nil)

		// マルチパートフォームの作成
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "receipt.png")
		_, _ = part.Write([]byte("test receipt image data"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/receipt", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleReceiptAnalyze(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
		}

		var response VisionResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success = true")
		}

		if response.Text == "" {
			t.Error("Expected non-empty text")
		}

		if response.Tokens == nil {
			t.Fatal("Expected non-nil tokens")
		}
	})

	t.Run("異常系_画像ファイルなし", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/receipt", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleReceiptAnalyze(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("異常系_認識エラー", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{
			RecognizeReceiptFunc: func(data []byte) (*entity.AIResult, error) {
				return nil, errors.New("receipt recognition failed")
			},
		}

		handler := NewVisionHandler(mockAI, nil, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "receipt.png")
		_, _ = part.Write([]byte("test image"))
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/receipt", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		rec := httptest.NewRecorder()

		handler.HandleReceiptAnalyze(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("異常系_GETメソッド", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/vision/receipt", nil)
		rec := httptest.NewRecorder()

		handler.HandleReceiptAnalyze(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestVisionHandler_HandleCategorize(t *testing.T) {
	t.Run("正常系_カテゴリ判定成功", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{
			CategorizeReceiptFunc: func(info string) (*entity.AIResult, error) {
				return &entity.AIResult{
					OriginalText:  info,
					CorrectedText: `{"category":"食費","confidence":0.95,"reason":"スーパーマーケットでの購入"}`,
					InputTokens:   50,
					OutputTokens:  30,
				}, nil
			},
		}

		handler := NewVisionHandler(mockAI, nil, nil)

		requestBody := map[string]string{
			"receipt_info": `{"store_name":"スーパーマーケット","items":[{"name":"野菜"}]}`,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/categorize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.HandleCategorize(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusOK)
		}

		var response VisionResponse
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success = true")
		}

		if response.Text == "" {
			t.Error("Expected non-empty text")
		}
	})

	t.Run("異常系_空のreceipt_info", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		requestBody := map[string]string{
			"receipt_info": "",
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/categorize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.HandleCategorize(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("異常系_不正なJSON", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/categorize", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.HandleCategorize(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("異常系_カテゴリ判定エラー", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{
			CategorizeReceiptFunc: func(info string) (*entity.AIResult, error) {
				return nil, errors.New("categorization failed")
			},
		}

		handler := NewVisionHandler(mockAI, nil, nil)

		requestBody := map[string]string{
			"receipt_info": `{"store_name":"test"}`,
		}
		body, _ := json.Marshal(requestBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/categorize", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.HandleCategorize(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusInternalServerError)
		}
	})

	t.Run("異常系_GETメソッド", func(t *testing.T) {
		mockAI := &MockAICorrectionUseCaseForVision{}
		handler := NewVisionHandler(mockAI, nil, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/vision/categorize", nil)
		rec := httptest.NewRecorder()

		handler.HandleCategorize(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("Status = %v, want %v", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}
