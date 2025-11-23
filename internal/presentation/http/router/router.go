package router

import (
	"net/http"

	"vision-api-app/internal/presentation/di"
	"vision-api-app/internal/presentation/http/middleware"
)

// NewRouter 新しいルーターを作成
func NewRouter(container *di.Container) http.Handler {
	mux := http.NewServeMux()

	// Web UI ハンドラー
	webHandler := container.WebHandler()
	mux.HandleFunc("/", webHandler.HandleUploadPage)
	mux.HandleFunc("/upload", webHandler.HandleUpload)
	mux.HandleFunc("/result", webHandler.HandleResult)
	mux.HandleFunc("/household", webHandler.HandleHousehold)

	// Static files
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Vision API ハンドラー
	visionHandler := container.VisionHandler()
	mux.HandleFunc("/api/v1/vision/analyze", visionHandler.HandleAnalyze)
	mux.HandleFunc("/api/v1/vision/receipt", visionHandler.HandleReceiptAnalyze)
	mux.HandleFunc("/api/v1/vision/categorize", visionHandler.HandleCategorize)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","version":"3.0.0"}`))
	})

	// ミドルウェアの適用
	var h http.Handler = mux
	h = middleware.Recovery(h)
	h = middleware.LoggerWithHealthCheck(h)
	h = middleware.CORS(h)

	return h
}
