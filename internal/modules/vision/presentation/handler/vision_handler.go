package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vision-api-app/internal/modules/household/domain/repository"
	"vision-api-app/internal/modules/vision/usecase"
)

// VisionHandler Vision API処理のハンドラー
type VisionHandler struct {
	aiCorrectionUseCase *usecase.AICorrectionUseCase
	cacheRepo           repository.CacheRepository
}

// NewVisionHandler 新しいVisionHandlerを作成
func NewVisionHandler(
	aiCorrectionUseCase *usecase.AICorrectionUseCase,
	cacheRepo repository.CacheRepository,
) *VisionHandler {
	return &VisionHandler{
		aiCorrectionUseCase: aiCorrectionUseCase,
		cacheRepo:           cacheRepo,
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
