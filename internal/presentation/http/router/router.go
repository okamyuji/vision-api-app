package router

import (
	"net/http"

	"vision-api-app/internal/presentation/di"
	"vision-api-app/internal/presentation/http/handler"
	"vision-api-app/internal/presentation/http/middleware"
)

// NewRouter 新しいルーターを作成
func NewRouter(container *di.Container) http.Handler {
	mux := http.NewServeMux()

	// ハンドラーの作成
	healthHandler := handler.NewHealthHandler()
	visionHandler := handler.NewVisionHandler(
		container.AICorrectionUseCase(),
		container.CacheRepository(),
		container.ReceiptRepository(),
	)

	// ルーティング
	mux.Handle("/health", healthHandler)
	mux.HandleFunc("/api/v1/vision/analyze", visionHandler.HandleAnalyze)
	mux.HandleFunc("/api/v1/vision/receipt", visionHandler.HandleReceiptAnalyze)
	mux.HandleFunc("/api/v1/vision/categorize", visionHandler.HandleCategorize)

	// ミドルウェアの適用
	var h http.Handler = mux
	h = middleware.Recovery(h)
	h = middleware.LoggerWithHealthCheck(h) // ヘルスチェックを除外
	h = middleware.CORS(h)

	return h
}
