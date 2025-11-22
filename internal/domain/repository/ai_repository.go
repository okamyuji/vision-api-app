package repository

import "vision-api-app/internal/domain/entity"

// AIRepository AI補正のリポジトリインターフェース
type AIRepository interface {
	// Correct テキストを補正（汎用）
	Correct(text string) (*entity.AIResult, error)

	// RecognizeImage 画像から直接テキストを認識（汎用）
	RecognizeImage(imageData []byte) (*entity.AIResult, error)

	// RecognizeReceipt レシート画像から構造化データを抽出
	RecognizeReceipt(imageData []byte) (*entity.AIResult, error)

	// CategorizeReceipt レシート情報から適切なカテゴリを判定
	CategorizeReceipt(receiptInfo string) (*entity.AIResult, error)

	// ProviderName プロバイダー名を返す
	ProviderName() string
}
