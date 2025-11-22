package usecase

import (
	"errors"
	"testing"

	"vision-api-app/internal/domain/entity"
)

// MockAIRepository AIRepositoryのモック
type MockAIRepository struct {
	CorrectFunc           func(string) (*entity.AIResult, error)
	RecognizeImageFunc    func([]byte) (*entity.AIResult, error)
	RecognizeReceiptFunc  func([]byte) (*entity.AIResult, error)
	CategorizeReceiptFunc func(string) (*entity.AIResult, error)
	ProviderNameFunc      func() string
}

func (m *MockAIRepository) Correct(text string) (*entity.AIResult, error) {
	if m.CorrectFunc != nil {
		return m.CorrectFunc(text)
	}
	return entity.NewAIResult(text, "corrected "+text, 10, 15, "test-model"), nil
}

func (m *MockAIRepository) RecognizeImage(imageData []byte) (*entity.AIResult, error) {
	if m.RecognizeImageFunc != nil {
		return m.RecognizeImageFunc(imageData)
	}
	return entity.NewAIResult("", "recognized text", 100, 50, "test-model"), nil
}

func (m *MockAIRepository) RecognizeReceipt(imageData []byte) (*entity.AIResult, error) {
	if m.RecognizeReceiptFunc != nil {
		return m.RecognizeReceiptFunc(imageData)
	}
	return entity.NewAIResult("", `{"store_name":"テストストア","total_amount":1500}`, 100, 50, "test-model"), nil
}

func (m *MockAIRepository) CategorizeReceipt(receiptInfo string) (*entity.AIResult, error) {
	if m.CategorizeReceiptFunc != nil {
		return m.CategorizeReceiptFunc(receiptInfo)
	}
	return entity.NewAIResult(receiptInfo, `{"category":"食費","confidence":0.95}`, 50, 30, "test-model"), nil
}

func (m *MockAIRepository) ProviderName() string {
	if m.ProviderNameFunc != nil {
		return m.ProviderNameFunc()
	}
	return "Test AI"
}

func TestAICorrectionUseCase_Correct(t *testing.T) {
	mockRepo := &MockAIRepository{
		CorrectFunc: func(text string) (*entity.AIResult, error) {
			return entity.NewAIResult(text, "corrected text", 10, 15, "model"), nil
		},
	}

	useCase := NewAICorrectionUseCase(mockRepo)
	result, err := useCase.Correct("test text")

	if err != nil {
		t.Fatalf("Correct() error = %v", err)
	}

	if result.OriginalText != "test text" {
		t.Errorf("Expected original text 'test text', got '%s'", result.OriginalText)
	}

	if result.CorrectedText != "corrected text" {
		t.Errorf("Expected corrected text 'corrected text', got '%s'", result.CorrectedText)
	}
}

func TestAICorrectionUseCase_Correct_EmptyText(t *testing.T) {
	mockRepo := &MockAIRepository{}
	useCase := NewAICorrectionUseCase(mockRepo)

	_, err := useCase.Correct("")
	if err == nil {
		t.Error("Expected error for empty text, got nil")
	}

	_, err = useCase.Correct("   ")
	if err == nil {
		t.Error("Expected error for whitespace-only text, got nil")
	}
}

func TestAICorrectionUseCase_Correct_Error(t *testing.T) {
	mockRepo := &MockAIRepository{
		CorrectFunc: func(text string) (*entity.AIResult, error) {
			return nil, errors.New("API error")
		},
	}

	useCase := NewAICorrectionUseCase(mockRepo)
	_, err := useCase.Correct("test")

	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestAICorrectionUseCase_GetProviderName(t *testing.T) {
	mockRepo := &MockAIRepository{
		ProviderNameFunc: func() string {
			return "Anthropic Claude"
		},
	}

	useCase := NewAICorrectionUseCase(mockRepo)
	provider := useCase.GetProviderName()

	if provider != "Anthropic Claude" {
		t.Errorf("Expected provider 'Anthropic Claude', got '%s'", provider)
	}
}

func TestAICorrectionUseCase_RecognizeImage(t *testing.T) {
	tests := []struct {
		name          string
		imageData     []byte
		mockFunc      func([]byte) (*entity.AIResult, error)
		wantErr       bool
		expectedText  string
		expectedInput int
	}{
		{
			name:      "正常系: 画像認識成功",
			imageData: []byte("fake image data"),
			mockFunc: func(data []byte) (*entity.AIResult, error) {
				return entity.NewAIResult("", "recognized text from image", 100, 50, "claude-vision"), nil
			},
			wantErr:       false,
			expectedText:  "recognized text from image",
			expectedInput: 100,
		},
		{
			name:      "正常系: 日本語テキスト認識",
			imageData: []byte("fake japanese image"),
			mockFunc: func(data []byte) (*entity.AIResult, error) {
				return entity.NewAIResult("", "こんにちは世界", 150, 75, "claude-vision"), nil
			},
			wantErr:       false,
			expectedText:  "こんにちは世界",
			expectedInput: 150,
		},
		{
			name:      "異常系: 空の画像データ",
			imageData: []byte{},
			mockFunc:  nil,
			wantErr:   true,
		},
		{
			name:      "異常系: nil画像データ",
			imageData: nil,
			mockFunc:  nil,
			wantErr:   true,
		},
		{
			name:      "異常系: API エラー",
			imageData: []byte("fake image"),
			mockFunc: func(data []byte) (*entity.AIResult, error) {
				return nil, errors.New("API rate limit exceeded")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				RecognizeImageFunc: tt.mockFunc,
			}

			useCase := NewAICorrectionUseCase(mockRepo)
			result, err := useCase.RecognizeImage(tt.imageData)

			if (err != nil) != tt.wantErr {
				t.Errorf("RecognizeImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.CorrectedText != tt.expectedText {
					t.Errorf("Expected text '%s', got '%s'", tt.expectedText, result.CorrectedText)
				}

				if result.InputTokens != tt.expectedInput {
					t.Errorf("Expected input tokens %d, got %d", tt.expectedInput, result.InputTokens)
				}
			}
		})
	}
}

func TestAICorrectionUseCase_RecognizeImage_LargeImage(t *testing.T) {
	// 大きな画像データ（1MB）
	largeImage := make([]byte, 1024*1024)
	for i := range largeImage {
		largeImage[i] = byte(i % 256)
	}

	mockRepo := &MockAIRepository{
		RecognizeImageFunc: func(data []byte) (*entity.AIResult, error) {
			if len(data) != len(largeImage) {
				t.Errorf("Expected image size %d, got %d", len(largeImage), len(data))
			}
			return entity.NewAIResult("", "large image recognized", 500, 200, "claude-vision"), nil
		},
	}

	useCase := NewAICorrectionUseCase(mockRepo)
	result, err := useCase.RecognizeImage(largeImage)

	if err != nil {
		t.Fatalf("RecognizeImage() error = %v", err)
	}

	if result.CorrectedText != "large image recognized" {
		t.Errorf("Expected 'large image recognized', got '%s'", result.CorrectedText)
	}
}

func TestAICorrectionUseCase_Correct_LongText(t *testing.T) {
	// 長いテキスト（10KB）
	longText := ""
	for i := 0; i < 1000; i++ {
		longText += "This is a test sentence. "
	}

	mockRepo := &MockAIRepository{
		CorrectFunc: func(text string) (*entity.AIResult, error) {
			return entity.NewAIResult(text, "corrected long text", 5000, 5100, "model"), nil
		},
	}

	useCase := NewAICorrectionUseCase(mockRepo)
	result, err := useCase.Correct(longText)

	if err != nil {
		t.Fatalf("Correct() error = %v", err)
	}

	if result.OriginalText != longText {
		t.Error("Original text mismatch")
	}

	if result.CorrectedText != "corrected long text" {
		t.Errorf("Expected 'corrected long text', got '%s'", result.CorrectedText)
	}
}

func TestAICorrectionUseCase_Correct_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "特殊文字: 改行",
			text: "line1\nline2\nline3",
		},
		{
			name: "特殊文字: タブ",
			text: "col1\tcol2\tcol3",
		},
		{
			name: "特殊文字: Unicode",
			text: "Hello 世界 🌍",
		},
		{
			name: "特殊文字: エスケープ文字",
			text: "quote: \"test\" backslash: \\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				CorrectFunc: func(text string) (*entity.AIResult, error) {
					return entity.NewAIResult(text, "corrected: "+text, 10, 15, "model"), nil
				},
			}

			useCase := NewAICorrectionUseCase(mockRepo)
			result, err := useCase.Correct(tt.text)

			if err != nil {
				t.Fatalf("Correct() error = %v", err)
			}

			if result.OriginalText != tt.text {
				t.Errorf("Original text mismatch")
			}
		})
	}
}

func TestAICorrectionUseCase_GetProviderName_Default(t *testing.T) {
	// デフォルトのProviderNameFunc
	mockRepo := &MockAIRepository{}
	useCase := NewAICorrectionUseCase(mockRepo)
	provider := useCase.GetProviderName()

	if provider != "Test AI" {
		t.Errorf("Expected default provider 'Test AI', got '%s'", provider)
	}
}

func TestNewAICorrectionUseCase(t *testing.T) {
	mockRepo := &MockAIRepository{}
	useCase := NewAICorrectionUseCase(mockRepo)

	if useCase == nil {
		t.Fatal("NewAICorrectionUseCase() returned nil")
	}

	// useCaseが正しく初期化されているか確認
	provider := useCase.GetProviderName()
	if provider == "" {
		t.Error("Provider name is empty")
	}
}

func TestAICorrectionUseCase_Correct_WhitespaceVariations(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{
			name:    "異常系: 空文字列",
			text:    "",
			wantErr: true,
		},
		{
			name:    "異常系: スペースのみ",
			text:    "   ",
			wantErr: true,
		},
		{
			name:    "異常系: タブのみ",
			text:    "\t\t\t",
			wantErr: true,
		},
		{
			name:    "異常系: 改行のみ",
			text:    "\n\n\n",
			wantErr: true,
		},
		{
			name:    "正常系: 前後にスペース",
			text:    "  test  ",
			wantErr: false,
		},
		{
			name:    "正常系: 1文字",
			text:    "a",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				CorrectFunc: func(text string) (*entity.AIResult, error) {
					return entity.NewAIResult(text, "corrected", 10, 15, "model"), nil
				},
			}

			useCase := NewAICorrectionUseCase(mockRepo)
			_, err := useCase.Correct(tt.text)

			if (err != nil) != tt.wantErr {
				t.Errorf("Correct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAICorrectionUseCase_RecognizeImage_ErrorMessages(t *testing.T) {
	tests := []struct {
		name      string
		imageData []byte
		mockError error
		wantError string
	}{
		{
			name:      "異常系: 空画像データ",
			imageData: []byte{},
			mockError: nil,
			wantError: "image data is empty",
		},
		{
			name:      "異常系: API認証エラー",
			imageData: []byte("test"),
			mockError: errors.New("authentication failed"),
			wantError: "claude vision ocr processing failed",
		},
		{
			name:      "異常系: ネットワークエラー",
			imageData: []byte("test"),
			mockError: errors.New("network timeout"),
			wantError: "claude vision ocr processing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				RecognizeImageFunc: func(data []byte) (*entity.AIResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return entity.NewAIResult("", "text", 10, 5, "model"), nil
				},
			}

			useCase := NewAICorrectionUseCase(mockRepo)
			_, err := useCase.RecognizeImage(tt.imageData)

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !contains(err.Error(), tt.wantError) {
				t.Errorf("Error message '%s' does not contain '%s'", err.Error(), tt.wantError)
			}
		})
	}
}

func TestAICorrectionUseCase_Correct_ErrorMessages(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		mockError error
		wantError string
	}{
		{
			name:      "異常系: 空テキスト",
			text:      "",
			mockError: nil,
			wantError: "text is empty",
		},
		{
			name:      "異常系: APIエラー",
			text:      "test",
			mockError: errors.New("rate limit exceeded"),
			wantError: "AI correction failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				CorrectFunc: func(text string) (*entity.AIResult, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return entity.NewAIResult(text, "corrected", 10, 15, "model"), nil
				},
			}

			useCase := NewAICorrectionUseCase(mockRepo)
			_, err := useCase.Correct(tt.text)

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !contains(err.Error(), tt.wantError) {
				t.Errorf("Error message '%s' does not contain '%s'", err.Error(), tt.wantError)
			}
		})
	}
}

func TestAICorrectionUseCase_RecognizeReceipt(t *testing.T) {
	tests := []struct {
		name      string
		imageData []byte
		mockFunc  func([]byte) (*entity.AIResult, error)
		wantErr   bool
	}{
		{
			name:      "正常系: レシート認識成功",
			imageData: []byte("test receipt image"),
			mockFunc: func(data []byte) (*entity.AIResult, error) {
				return entity.NewAIResult("", `{"store_name":"スーパーマーケット","total_amount":1500}`, 100, 50, "model"), nil
			},
			wantErr: false,
		},
		{
			name:      "異常系: 空の画像データ",
			imageData: []byte{},
			wantErr:   true,
		},
		{
			name:      "異常系: リポジトリエラー",
			imageData: []byte("test image"),
			mockFunc: func(data []byte) (*entity.AIResult, error) {
				return nil, errors.New("receipt recognition failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				RecognizeReceiptFunc: tt.mockFunc,
			}

			useCase := NewAICorrectionUseCase(mockRepo)
			result, err := useCase.RecognizeReceipt(tt.imageData)

			if (err != nil) != tt.wantErr {
				t.Errorf("RecognizeReceipt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}

			if !tt.wantErr && result != nil {
				if result.CorrectedText == "" {
					t.Error("Expected non-empty corrected text")
				}
			}
		})
	}
}

func TestAICorrectionUseCase_CategorizeReceipt(t *testing.T) {
	tests := []struct {
		name        string
		receiptInfo string
		mockFunc    func(string) (*entity.AIResult, error)
		wantErr     bool
	}{
		{
			name:        "正常系: カテゴリ判定成功",
			receiptInfo: `{"store_name":"スーパーマーケット","items":[{"name":"野菜"}]}`,
			mockFunc: func(info string) (*entity.AIResult, error) {
				return entity.NewAIResult(info, `{"category":"食費","confidence":0.95}`, 50, 30, "model"), nil
			},
			wantErr: false,
		},
		{
			name:        "異常系: 空のレシート情報",
			receiptInfo: "",
			wantErr:     true,
		},
		{
			name:        "異常系: 空白のみ",
			receiptInfo: "   ",
			wantErr:     true,
		},
		{
			name:        "異常系: リポジトリエラー",
			receiptInfo: `{"store_name":"test"}`,
			mockFunc: func(info string) (*entity.AIResult, error) {
				return nil, errors.New("categorization failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				CategorizeReceiptFunc: tt.mockFunc,
			}

			useCase := NewAICorrectionUseCase(mockRepo)
			result, err := useCase.CategorizeReceipt(tt.receiptInfo)

			if (err != nil) != tt.wantErr {
				t.Errorf("CategorizeReceipt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}

			if !tt.wantErr && result != nil {
				if result.CorrectedText == "" {
					t.Error("Expected non-empty corrected text")
				}
			}
		})
	}
}

// contains ヘルパー関数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
