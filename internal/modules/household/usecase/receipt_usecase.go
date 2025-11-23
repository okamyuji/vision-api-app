package usecase

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"vision-api-app/internal/modules/household/domain/entity"
	"vision-api-app/internal/modules/household/domain/repository"
	"vision-api-app/internal/modules/vision/domain"
)

// ReceiptUseCase レシート処理のユースケース
type ReceiptUseCase struct {
	aiRepo      domain.AIRepository
	receiptRepo repository.ReceiptRepository
}

// NewReceiptUseCase 新しいReceiptUseCaseを作成
func NewReceiptUseCase(aiRepo domain.AIRepository, receiptRepo repository.ReceiptRepository) *ReceiptUseCase {
	return &ReceiptUseCase{
		aiRepo:      aiRepo,
		receiptRepo: receiptRepo,
	}
}

// ProcessReceiptImage レシート画像を処理してデータベースに保存
func (uc *ReceiptUseCase) ProcessReceiptImage(ctx context.Context, imageData []byte) (*entity.Receipt, error) {
	// 画像からレシート情報を抽出
	aiResult, err := uc.aiRepo.RecognizeReceipt(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to recognize receipt: %w", err)
	}

	// JSONをパース
	receipt, err := uc.parseReceiptJSON(aiResult.CorrectedText)
	if err != nil {
		return nil, fmt.Errorf("failed to parse receipt JSON: %w", err)
	}

	// データベースに保存
	if err := uc.receiptRepo.Create(ctx, receipt); err != nil {
		return nil, fmt.Errorf("failed to save receipt: %w", err)
	}

	return receipt, nil
}

// GetReceipt レシートを取得
func (uc *ReceiptUseCase) GetReceipt(ctx context.Context, id string) (*entity.Receipt, error) {
	return uc.receiptRepo.FindByID(ctx, id)
}

// ListReceipts レシート一覧を取得
func (uc *ReceiptUseCase) ListReceipts(ctx context.Context, limit, offset int) ([]*entity.Receipt, error) {
	return uc.receiptRepo.FindAll(ctx, limit, offset)
}

// parseReceiptJSON JSONからレシートエンティティを作成
func (uc *ReceiptUseCase) parseReceiptJSON(receiptJSON string) (*entity.Receipt, error) {
	// Claude APIは```json```で囲まれた形式で返すことがあるため、クリーンアップ
	cleanJSON := receiptJSON
	if idx := bytes.Index([]byte(receiptJSON), []byte("```json")); idx != -1 {
		cleanJSON = receiptJSON[idx+7:]
		if idx := bytes.Index([]byte(cleanJSON), []byte("```")); idx != -1 {
			cleanJSON = cleanJSON[:idx]
		}
	}
	cleanJSONBytes := bytes.TrimSpace([]byte(cleanJSON))

	var receiptData struct {
		StoreName     string `json:"store_name"`
		PurchaseDate  string `json:"purchase_date"`
		TotalAmount   int    `json:"total_amount"`
		TaxAmount     int    `json:"tax_amount"`
		PaymentMethod string `json:"payment_method"`
		ReceiptNumber string `json:"receipt_number"`
		Items         []struct {
			Name     string `json:"name"`
			Quantity int    `json:"quantity"`
			Price    int    `json:"price"`
		} `json:"items"`
	}

	if err := json.Unmarshal(cleanJSONBytes, &receiptData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// 【重要】total_amountをitemsの合計で強制的に上書き
	calculatedTotal := 0
	for _, item := range receiptData.Items {
		calculatedTotal += item.Price * item.Quantity
	}
	if calculatedTotal > 0 {
		receiptData.TotalAmount = calculatedTotal
	}

	// 購入日時のパース
	var purchaseDate time.Time
	if receiptData.PurchaseDate != "" {
		formats := []string{
			"2006-01-02 15:04",
			"2006-01-02",
			"2006/01/02 15:04",
			"2006/01/02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, receiptData.PurchaseDate); err == nil {
				purchaseDate = t
				break
			}
		}
	}
	if purchaseDate.IsZero() {
		purchaseDate = time.Now()
	}

	// レシートエンティティの作成
	receipt := &entity.Receipt{
		ID:            generateUUID(),
		StoreName:     receiptData.StoreName,
		PurchaseDate:  purchaseDate,
		TotalAmount:   receiptData.TotalAmount,
		TaxAmount:     receiptData.TaxAmount,
		PaymentMethod: receiptData.PaymentMethod,
		ReceiptNumber: receiptData.ReceiptNumber,
		Category:      "",
		Items:         make([]entity.ReceiptItem, 0, len(receiptData.Items)),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 商品アイテムの追加
	for _, item := range receiptData.Items {
		if item.Name != "" {
			receiptItem := entity.ReceiptItem{
				ID:        generateUUID(),
				ReceiptID: receipt.ID,
				Name:      item.Name,
				Quantity:  item.Quantity,
				Price:     item.Price,
				CreatedAt: time.Now(),
			}
			receipt.Items = append(receipt.Items, receiptItem)
		}
	}

	return receipt, nil
}

// generateUUID UUIDを生成（簡易版）
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
