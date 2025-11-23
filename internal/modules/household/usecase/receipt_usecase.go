package usecase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"vision-api-app/internal/modules/household/domain/entity"
	"vision-api-app/internal/modules/household/domain/repository"
	"vision-api-app/internal/modules/vision/domain"
)

// ReceiptUseCase レシート処理のユースケース
type ReceiptUseCase struct {
	aiRepo      domain.AIRepository
	receiptRepo repository.ReceiptRepository
	cacheRepo   repository.CacheRepository
}

// NewReceiptUseCase 新しいReceiptUseCaseを作成
func NewReceiptUseCase(aiRepo domain.AIRepository, receiptRepo repository.ReceiptRepository, cacheRepo repository.CacheRepository) *ReceiptUseCase {
	return &ReceiptUseCase{
		aiRepo:      aiRepo,
		receiptRepo: receiptRepo,
		cacheRepo:   cacheRepo,
	}
}

// ProcessReceiptImage レシート画像を処理してデータベースに保存
func (uc *ReceiptUseCase) ProcessReceiptImage(ctx context.Context, imageData []byte) (*entity.Receipt, error) {
	// キャッシュキーの生成（画像データのSHA256ハッシュ）
	cacheKey := uc.generateCacheKey("receipt", imageData)

	// キャッシュチェック
	var receiptJSON string
	if uc.cacheRepo != nil {
		if cached, err := uc.cacheRepo.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			receiptJSON = string(cached)
		}
	}

	// キャッシュミスの場合、AI APIを呼び出す
	if receiptJSON == "" {
		aiResult, err := uc.aiRepo.RecognizeReceipt(imageData)
		if err != nil {
			return nil, fmt.Errorf("failed to recognize receipt: %w", err)
		}
		receiptJSON = aiResult.CorrectedText

		// キャッシュに保存（24時間）
		if uc.cacheRepo != nil {
			_ = uc.cacheRepo.Set(ctx, cacheKey, []byte(receiptJSON), 24*time.Hour)
		}
	}

	// 画像ハッシュから一意のレシートIDを生成
	receiptID := uc.generateReceiptID(imageData)

	// 既存のレシートをチェック
	existingReceipt, err := uc.receiptRepo.FindByID(ctx, receiptID)
	if err == nil && existingReceipt != nil {
		// 既に同じ画像のレシートが存在する場合は、それを返す
		return existingReceipt, nil
	}

	// JSONをパース（IDを渡してパース時に設定）
	receipt, err := uc.parseReceiptJSON(receiptJSON, receiptID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse receipt JSON: %w", err)
	}

	// 明細項目ごとにカテゴリーを判定
	// カテゴリー判定エラーは致命的ではないので無視
	_ = uc.categorizeReceiptItems(receipt)

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
func (uc *ReceiptUseCase) parseReceiptJSON(receiptJSON string, receiptID string) (*entity.Receipt, error) {
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
		ID:            receiptID,
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
	for i, item := range receiptData.Items {
		if item.Name != "" {
			// アイテムIDはレシートIDの先頭27文字 + "-" + インデックス（8桁）でUUID形式（36文字）を保証
			// 例: b5377e40-a9f1-4426-6dfe-bd1-00000000
			itemID := fmt.Sprintf("%s-%08d", receiptID[:27], i)
			receiptItem := entity.ReceiptItem{
				ID:        itemID,
				ReceiptID: receiptID,
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

// generateCacheKey キャッシュキーを生成
func (uc *ReceiptUseCase) generateCacheKey(prefix string, data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("vision:%s:%s", prefix, hex.EncodeToString(hash[:]))
}

// generateReceiptID 画像データからレシートIDを生成（UUID形式）
func (uc *ReceiptUseCase) generateReceiptID(imageData []byte) string {
	hash := sha256.Sum256(imageData)
	// UUID v4形式に変換（8-4-4-4-12 = 36文字）
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		hash[0:4],
		hash[4:6],
		hash[6:8],
		hash[8:10],
		hash[10:16])
}

// categorizeReceiptItems 明細項目ごとにカテゴリーを判定
func (uc *ReceiptUseCase) categorizeReceiptItems(receipt *entity.Receipt) error {
	if len(receipt.Items) == 0 {
		return nil
	}

	// 商品名リストを作成
	itemNames := make([]string, len(receipt.Items))
	for i, item := range receipt.Items {
		itemNames[i] = item.Name
	}

	// AI APIで一括カテゴリー判定
	itemsInfo := fmt.Sprintf("店名: %s\n以下の商品それぞれのカテゴリーを判定してください（食費、日用品、医療費、娯楽費、交通費、通信費、光熱費、その他）:\n", receipt.StoreName)
	for i, name := range itemNames {
		itemsInfo += fmt.Sprintf("%d. %s\n", i+1, name)
	}

	result, err := uc.aiRepo.CategorizeReceipt(itemsInfo)
	if err != nil {
		return fmt.Errorf("failed to categorize items: %w", err)
	}

	// レスポンスをパース
	categories, err := uc.parseItemCategories(result.CorrectedText, len(receipt.Items))
	if err != nil {
		return fmt.Errorf("failed to parse categories: %w", err)
	}

	// 各明細項目にカテゴリーを設定
	for i := range receipt.Items {
		if i < len(categories) && categories[i] != "" {
			receipt.Items[i].Category = categories[i]
		} else {
			receipt.Items[i].Category = "その他"
		}
	}

	return nil
}

// parseItemCategories AI APIのレスポンスから商品ごとのカテゴリーを抽出
func (uc *ReceiptUseCase) parseItemCategories(response string, itemCount int) ([]string, error) {
	// ```json で囲まれている場合は抽出
	cleanResponse := response
	if idx := bytes.Index([]byte(response), []byte("```json")); idx != -1 {
		cleanResponse = response[idx+7:]
		if idx := bytes.Index([]byte(cleanResponse), []byte("```")); idx != -1 {
			cleanResponse = cleanResponse[:idx]
		}
	}
	cleanBytes := bytes.TrimSpace([]byte(cleanResponse))

	// JSON配列形式を試す: ["食費", "日用品", ...]
	var categoriesArray []string
	if err := json.Unmarshal(cleanBytes, &categoriesArray); err == nil {
		return categoriesArray, nil
	}

	// オブジェクト配列形式を試す: [{"item": "商品名", "category": "食費", ...}, ...]
	var itemObjects []struct {
		Item     string `json:"item"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(cleanBytes, &itemObjects); err == nil && len(itemObjects) > 0 {
		categories := make([]string, len(itemObjects))
		for i, obj := range itemObjects {
			categories[i] = obj.Category
		}
		return categories, nil
	}

	// 番号付きオブジェクト形式を先にチェック: {"1": "食費", "2": "日用品", ...}
	var categoriesMap map[string]string
	if err := json.Unmarshal(cleanBytes, &categoriesMap); err == nil {
		// "categories"キーがある場合は別の形式なのでスキップ
		if _, hasCategories := categoriesMap["categories"]; !hasCategories && len(categoriesMap) > 0 {
			// 数字キーが存在するかチェック
			categories := make([]string, 0, itemCount)
			for i := 0; i < itemCount; i++ {
				key := fmt.Sprintf("%d", i+1)
				if cat, ok := categoriesMap[key]; ok {
					categories = append(categories, cat)
				}
			}
			if len(categories) > 0 {
				return categories, nil
			}
		}
	}

	// JSONオブジェクト形式を試す: {"categories": ["食費", "日用品", ...]}
	var categoriesObj struct {
		Categories []string `json:"categories"`
	}
	if err := json.Unmarshal(cleanBytes, &categoriesObj); err == nil && len(categoriesObj.Categories) > 0 {
		return categoriesObj.Categories, nil
	}

	// プレーンテキスト形式を試す（改行区切り）
	lines := bytes.Split(cleanBytes, []byte("\n"))
	categories := make([]string, 0, itemCount)
	for _, line := range lines {
		lineStr := string(bytes.TrimSpace(line))
		if lineStr != "" {
			// "1. 食費" のような形式から抽出
			if idx := bytes.IndexByte(line, '.'); idx != -1 && idx < len(line)-1 {
				lineStr = string(bytes.TrimSpace(line[idx+1:]))
			}
			categories = append(categories, lineStr)
		}
	}

	if len(categories) > 0 {
		return categories, nil
	}

	return nil, fmt.Errorf("failed to parse categories from response")
}
