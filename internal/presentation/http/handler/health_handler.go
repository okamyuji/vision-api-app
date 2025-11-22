package handler

import (
	"encoding/json"
	"net/http"
)

// HealthHandler ヘルスチェックのハンドラー
type HealthHandler struct{}

// NewHealthHandler 新しいHealthHandlerを作成
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse ヘルスチェックのレスポンス
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// ServeHTTP ヘルスチェックを処理
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := HealthResponse{
		Status:  "ok",
		Version: "2.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}
