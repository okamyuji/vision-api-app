package usecase

import (
	"fmt"
	"strings"

	"vision-api-app/internal/modules/vision/domain"
)

// AICorrectionUseCase AI補正のユースケース
type AICorrectionUseCase struct {
	aiRepo domain.AIRepository
}

// NewAICorrectionUseCase 新しいAICorrectionUseCaseを作成
func NewAICorrectionUseCase(aiRepo domain.AIRepository) *AICorrectionUseCase {
	return &AICorrectionUseCase{
		aiRepo: aiRepo,
	}
}

// Correct テキストを補正
func (uc *AICorrectionUseCase) Correct(text string) (*domain.AIResult, error) {
	// 入力検証
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is empty")
	}

	// AI補正実行
	result, err := uc.aiRepo.Correct(text)
	if err != nil {
		return nil, fmt.Errorf("AI correction failed: %w", err)
	}

	return result, nil
}

// RecognizeImage 画像から直接テキストを認識（汎用）
func (uc *AICorrectionUseCase) RecognizeImage(imageData []byte) (*domain.AIResult, error) {
	// 入力検証
	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}

	// Claude Vision APIでOCR実行
	result, err := uc.aiRepo.RecognizeImage(imageData)
	if err != nil {
		return nil, fmt.Errorf("claude vision ocr processing failed: %w", err)
	}

	return result, nil
}

// RecognizeReceipt レシート画像から構造化データを抽出
func (uc *AICorrectionUseCase) RecognizeReceipt(imageData []byte) (*domain.AIResult, error) {
	// 入力検証
	if len(imageData) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}

	// Claude Vision APIでレシート認識実行
	result, err := uc.aiRepo.RecognizeReceipt(imageData)
	if err != nil {
		return nil, fmt.Errorf("receipt recognition failed: %w", err)
	}

	return result, nil
}

// CategorizeReceipt レシート情報から適切なカテゴリを判定
func (uc *AICorrectionUseCase) CategorizeReceipt(receiptInfo string) (*domain.AIResult, error) {
	// 入力検証
	if strings.TrimSpace(receiptInfo) == "" {
		return nil, fmt.Errorf("receipt info is empty")
	}

	// カテゴリ判定実行
	result, err := uc.aiRepo.CategorizeReceipt(receiptInfo)
	if err != nil {
		return nil, fmt.Errorf("receipt categorization failed: %w", err)
	}

	return result, nil
}

// GetProviderName プロバイダー名を取得
func (uc *AICorrectionUseCase) GetProviderName() string {
	return uc.aiRepo.ProviderName()
}
