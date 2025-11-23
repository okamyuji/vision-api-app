package usecase

import (
	"errors"
	"testing"

	"vision-api-app/internal/modules/vision/domain"
)

// MockAIRepository モックAIリポジトリ
type MockAIRepository struct {
	CorrectFunc           func(text string) (*domain.AIResult, error)
	RecognizeImageFunc    func(imageData []byte) (*domain.AIResult, error)
	RecognizeReceiptFunc  func(imageData []byte) (*domain.AIResult, error)
	CategorizeReceiptFunc func(receiptInfo string) (*domain.AIResult, error)
	ProviderNameFunc      func() string
}

func (m *MockAIRepository) Correct(text string) (*domain.AIResult, error) {
	if m.CorrectFunc != nil {
		return m.CorrectFunc(text)
	}
	return domain.NewAIResult(text, "corrected", 10, 5, "test"), nil
}

func (m *MockAIRepository) RecognizeImage(imageData []byte) (*domain.AIResult, error) {
	if m.RecognizeImageFunc != nil {
		return m.RecognizeImageFunc(imageData)
	}
	return domain.NewAIResult("", "recognized text", 10, 5, "test"), nil
}

func (m *MockAIRepository) RecognizeReceipt(imageData []byte) (*domain.AIResult, error) {
	if m.RecognizeReceiptFunc != nil {
		return m.RecognizeReceiptFunc(imageData)
	}
	return domain.NewAIResult("", `{"store_name":"Test Store"}`, 10, 5, "test"), nil
}

func (m *MockAIRepository) CategorizeReceipt(receiptInfo string) (*domain.AIResult, error) {
	if m.CategorizeReceiptFunc != nil {
		return m.CategorizeReceiptFunc(receiptInfo)
	}
	return domain.NewAIResult(receiptInfo, `{"category":"食費"}`, 10, 5, "test"), nil
}

func (m *MockAIRepository) ProviderName() string {
	if m.ProviderNameFunc != nil {
		return m.ProviderNameFunc()
	}
	return "Mock AI Provider"
}

func TestNewAICorrectionUseCase(t *testing.T) {
	mockRepo := &MockAIRepository{}
	uc := NewAICorrectionUseCase(mockRepo)

	if uc == nil {
		t.Fatal("Expected non-nil usecase")
	}
	if uc.aiRepo != mockRepo {
		t.Error("Expected aiRepo to be set")
	}
}

func TestAICorrectionUseCase_Correct(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		mockErr error
		wantErr bool
	}{
		{
			name:    "正常なテキスト補正",
			text:    "test text",
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "空文字列",
			text:    "",
			mockErr: nil,
			wantErr: true,
		},
		{
			name:    "空白のみ",
			text:    "   ",
			mockErr: nil,
			wantErr: true,
		},
		{
			name:    "AIリポジトリエラー",
			text:    "test",
			mockErr: errors.New("AI error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				CorrectFunc: func(text string) (*domain.AIResult, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return domain.NewAIResult(text, "corrected", 10, 5, "test"), nil
				},
			}
			uc := NewAICorrectionUseCase(mockRepo)

			result, err := uc.Correct(tt.text)

			if (err != nil) != tt.wantErr {
				t.Errorf("Correct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestAICorrectionUseCase_RecognizeImage(t *testing.T) {
	tests := []struct {
		name      string
		imageData []byte
		mockErr   error
		wantErr   bool
	}{
		{
			name:      "正常な画像認識",
			imageData: []byte("image data"),
			mockErr:   nil,
			wantErr:   false,
		},
		{
			name:      "空の画像データ",
			imageData: []byte{},
			mockErr:   nil,
			wantErr:   true,
		},
		{
			name:      "nilの画像データ",
			imageData: nil,
			mockErr:   nil,
			wantErr:   true,
		},
		{
			name:      "AIリポジトリエラー",
			imageData: []byte("image data"),
			mockErr:   errors.New("AI error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				RecognizeImageFunc: func(imageData []byte) (*domain.AIResult, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return domain.NewAIResult("", "recognized", 10, 5, "test"), nil
				},
			}
			uc := NewAICorrectionUseCase(mockRepo)

			result, err := uc.RecognizeImage(tt.imageData)

			if (err != nil) != tt.wantErr {
				t.Errorf("RecognizeImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestAICorrectionUseCase_RecognizeReceipt(t *testing.T) {
	tests := []struct {
		name      string
		imageData []byte
		mockErr   error
		wantErr   bool
	}{
		{
			name:      "正常なレシート認識",
			imageData: []byte("receipt image"),
			mockErr:   nil,
			wantErr:   false,
		},
		{
			name:      "空の画像データ",
			imageData: []byte{},
			mockErr:   nil,
			wantErr:   true,
		},
		{
			name:      "AIリポジトリエラー",
			imageData: []byte("receipt image"),
			mockErr:   errors.New("AI error"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				RecognizeReceiptFunc: func(imageData []byte) (*domain.AIResult, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return domain.NewAIResult("", `{"store_name":"Test"}`, 10, 5, "test"), nil
				},
			}
			uc := NewAICorrectionUseCase(mockRepo)

			result, err := uc.RecognizeReceipt(tt.imageData)

			if (err != nil) != tt.wantErr {
				t.Errorf("RecognizeReceipt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestAICorrectionUseCase_CategorizeReceipt(t *testing.T) {
	tests := []struct {
		name        string
		receiptInfo string
		mockErr     error
		wantErr     bool
	}{
		{
			name:        "正常なカテゴリ判定",
			receiptInfo: `{"store_name":"スーパー"}`,
			mockErr:     nil,
			wantErr:     false,
		},
		{
			name:        "空文字列",
			receiptInfo: "",
			mockErr:     nil,
			wantErr:     true,
		},
		{
			name:        "空白のみ",
			receiptInfo: "   ",
			mockErr:     nil,
			wantErr:     true,
		},
		{
			name:        "AIリポジトリエラー",
			receiptInfo: `{"store_name":"スーパー"}`,
			mockErr:     errors.New("AI error"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockAIRepository{
				CategorizeReceiptFunc: func(receiptInfo string) (*domain.AIResult, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return domain.NewAIResult(receiptInfo, `{"category":"食費"}`, 10, 5, "test"), nil
				},
			}
			uc := NewAICorrectionUseCase(mockRepo)

			result, err := uc.CategorizeReceipt(tt.receiptInfo)

			if (err != nil) != tt.wantErr {
				t.Errorf("CategorizeReceipt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestAICorrectionUseCase_GetProviderName(t *testing.T) {
	mockRepo := &MockAIRepository{
		ProviderNameFunc: func() string {
			return "Test Provider"
		},
	}
	uc := NewAICorrectionUseCase(mockRepo)

	name := uc.GetProviderName()

	if name != "Test Provider" {
		t.Errorf("Expected 'Test Provider', got '%s'", name)
	}
}
