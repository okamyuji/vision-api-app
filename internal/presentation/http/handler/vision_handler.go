package handler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"vision-api-app/internal/domain/entity"
)

// VisionHandler Vision API処理のハンドラー
type VisionHandler struct {
	aiCorrectionUseCase AICorrectionUseCaseInterface
	cacheRepo           CacheRepositoryInterface
	receiptRepo         ReceiptRepositoryInterface
}

// NewVisionHandler 新しいVisionHandlerを作成
func NewVisionHandler(
	aiCorrectionUseCase AICorrectionUseCaseInterface,
	cacheRepo CacheRepositoryInterface,
	receiptRepo ReceiptRepositoryInterface,
) *VisionHandler {
	return &VisionHandler{
		aiCorrectionUseCase: aiCorrectionUseCase,
		cacheRepo:           cacheRepo,
		receiptRepo:         receiptRepo,
	}
}

// VisionResponse Vision APIレスポンス
type VisionResponse struct {
	Success bool              `json:"success"`
	Text    string            `json:"text"`
	Tokens  *AITokensResponse `json:"tokens,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// AITokensResponse AIトークン使用量のレスポンス
type AITokensResponse struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// HandleAnalyze 画像解析ハンドラー（汎用）
func (h *VisionHandler) HandleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// マルチパートフォームのパース
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB制限
		h.sendError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// 画像ファイルの取得
	file, _, err := r.FormFile("image")
	if err != nil {
		h.sendError(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// 画像データの読み込み
	imageData, err := io.ReadAll(file)
	if err != nil {
		h.sendError(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// キャッシュキーの生成
	cacheKey := h.generateCacheKey("analyze", imageData)

	// Redisキャッシュチェック
	if h.cacheRepo != nil {
		if cached, err := h.cacheRepo.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			response := VisionResponse{
				Success: true,
				Text:    string(cached),
				Tokens: &AITokensResponse{
					InputTokens:  0,
					OutputTokens: 0,
					TotalTokens:  0,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Claude Vision APIで画像解析
	aiResult, err := h.aiCorrectionUseCase.RecognizeImage(imageData)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Vision API failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Redisにキャッシュ保存（24時間）
	if h.cacheRepo != nil {
		_ = h.cacheRepo.Set(ctx, cacheKey, []byte(aiResult.CorrectedText), 24*time.Hour)
	}

	// レスポンスの構築
	response := VisionResponse{
		Success: true,
		Text:    aiResult.CorrectedText,
		Tokens: &AITokensResponse{
			InputTokens:  aiResult.InputTokens,
			OutputTokens: aiResult.OutputTokens,
			TotalTokens:  aiResult.TotalTokens(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// HandleReceiptAnalyze レシート画像解析ハンドラー
func (h *VisionHandler) HandleReceiptAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// マルチパートフォームのパース
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB制限
		h.sendError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// 画像ファイルの取得
	file, _, err := r.FormFile("image")
	if err != nil {
		h.sendError(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// 画像データの読み込み
	imageData, err := io.ReadAll(file)
	if err != nil {
		h.sendError(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// キャッシュキーの生成（画像データのハッシュ）
	cacheKey := h.generateCacheKey("receipt", imageData)

	// Redisキャッシュチェック
	if h.cacheRepo != nil {
		if cached, err := h.cacheRepo.Get(ctx, cacheKey); err == nil && len(cached) > 0 {
			// キャッシュヒット
			response := VisionResponse{
				Success: true,
				Text:    string(cached),
				Tokens: &AITokensResponse{
					InputTokens:  0,
					OutputTokens: 0,
					TotalTokens:  0,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(response)
			return
		}
	}

	// Claude Vision APIでレシート解析
	aiResult, err := h.aiCorrectionUseCase.RecognizeReceipt(imageData)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Receipt recognition failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Redisにキャッシュ保存（24時間）
	if h.cacheRepo != nil {
		_ = h.cacheRepo.Set(ctx, cacheKey, []byte(aiResult.CorrectedText), 24*time.Hour)
	}

	// MySQLにレシート保存（バックグラウンド処理）
	if h.receiptRepo != nil {
		fmt.Printf("Starting background save for receipt...\n")
		go h.saveReceiptToDatabase(context.Background(), aiResult.CorrectedText)
	} else {
		fmt.Printf("Receipt repository is nil, skipping save\n")
	}

	// レスポンスの構築
	response := VisionResponse{
		Success: true,
		Text:    aiResult.CorrectedText,
		Tokens: &AITokensResponse{
			InputTokens:  aiResult.InputTokens,
			OutputTokens: aiResult.OutputTokens,
			TotalTokens:  aiResult.TotalTokens(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// HandleCategorize カテゴリ判定ハンドラー
func (h *VisionHandler) HandleCategorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// リクエストボディの読み込み
	var request struct {
		ReceiptInfo string `json:"receipt_info"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ReceiptInfo == "" {
		h.sendError(w, "receipt_info is required", http.StatusBadRequest)
		return
	}

	// カテゴリ判定実行
	aiResult, err := h.aiCorrectionUseCase.CategorizeReceipt(request.ReceiptInfo)
	if err != nil {
		h.sendError(w, fmt.Sprintf("Categorization failed: %v", err), http.StatusInternalServerError)
		return
	}

	// レスポンスの構築
	response := VisionResponse{
		Success: true,
		Text:    aiResult.CorrectedText,
		Tokens: &AITokensResponse{
			InputTokens:  aiResult.InputTokens,
			OutputTokens: aiResult.OutputTokens,
			TotalTokens:  aiResult.TotalTokens(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// sendError エラーレスポンスを送信
func (h *VisionHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	response := VisionResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

// generateCacheKey キャッシュキーを生成
func (h *VisionHandler) generateCacheKey(prefix string, data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("vision:%s:%s", prefix, hex.EncodeToString(hash[:]))
}

// saveReceiptToDatabase レシートをデータベースに保存
func (h *VisionHandler) saveReceiptToDatabase(ctx context.Context, receiptJSON string) {
	// JSONからレシートデータをパース
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
		// パースエラーをログに記録
		fmt.Printf("Failed to parse receipt JSON: %v\n", err)
		fmt.Printf("JSON: %s\n", string(cleanJSONBytes))
		return
	}

	// 【重要】total_amountをitemsの合計で強制的に上書き
	// Claudeが正しく計算していない場合があるため、アプリケーション側で修正
	originalTotal := receiptData.TotalAmount
	calculatedTotal := 0
	for _, item := range receiptData.Items {
		calculatedTotal += item.Price * item.Quantity
	}
	if calculatedTotal > 0 && calculatedTotal != originalTotal {
		fmt.Printf("Correcting total_amount: %d → %d (diff: %d)\n", originalTotal, calculatedTotal, calculatedTotal-originalTotal)
		receiptData.TotalAmount = calculatedTotal
	}

	// 購入日時のパース
	var purchaseDate time.Time
	if receiptData.PurchaseDate != "" {
		// 複数の日付フォーマットを試行
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
		Category:      "", // カテゴリは別途判定
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

	// データベースに保存
	if err := h.receiptRepo.Create(ctx, receipt); err != nil {
		// エラーをログに記録
		fmt.Printf("Failed to save receipt to database: %v\n", err)
		return
	}
	fmt.Printf("Successfully saved receipt: %s (store: %s, amount: %d)\n", receipt.ID, receipt.StoreName, receipt.TotalAmount)
}

// generateUUID UUIDを生成（簡易版）
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
