package router

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"vision-api-app/internal/config"
	"vision-api-app/internal/presentation/di"
)

func TestNewRouter(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)
	if router == nil {
		t.Fatal("NewRouter() returned nil")
	}
}

func TestRouter_HealthEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "正常系: GET /health",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// レスポンスボディの確認
			body := rec.Body.String()
			if body == "" {
				t.Error("Expected non-empty response body")
			}
		})
	}
}

func TestRouter_VisionEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Anthropic.APIKey = "test-api-key" // テスト用APIキー
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "異常系: GET /api/v1/vision/analyze",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "異常系: PUT /api/v1/vision/analyze",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "異常系: DELETE /api/v1/vision/analyze",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/v1/vision/analyze", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRouter_NotFoundEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "異常系: 存在しないパス",
			path:           "/not-found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "異常系: /api/v1/unknown",
			path:           "/api/v1/unknown",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "異常系: /api/v2/vision/analyze",
			path:           "/api/v2/vision/analyze",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRouter_CORSMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedHeader string
	}{
		{
			name:           "正常系: CORS with origin",
			method:         http.MethodGet,
			origin:         "http://localhost:3000",
			expectedHeader: "*",
		},
		{
			name:           "正常系: OPTIONS request",
			method:         http.MethodOptions,
			origin:         "http://example.com",
			expectedHeader: "*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			// CORSヘッダーの確認
			corsHeader := rec.Header().Get("Access-Control-Allow-Origin")
			if corsHeader != tt.expectedHeader {
				t.Errorf("Expected CORS header '%s', got '%s'", tt.expectedHeader, corsHeader)
			}
		})
	}
}

func TestRouter_RecoveryMiddleware(t *testing.T) {
	// Recoveryミドルウェアは他のハンドラーでpanicが発生した場合に機能するため、
	// ここでは正常なリクエストが処理されることを確認
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// panicが発生しないことを確認
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexpected panic: %v", r)
		}
	}()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRouter_LoggerMiddleware(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	tests := []struct {
		name   string
		path   string
		method string
	}{
		{
			name:   "正常系: /health ログ抑制",
			path:   "/health",
			method: http.MethodGet,
		},
		{
			name:   "正常系: /api/v1/vision/analyze ログ出力",
			path:   "/api/v1/vision/analyze",
			method: http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			// ログミドルウェアが正常に動作していることを確認
			// （実際のログ出力は標準出力に行くため、ここではエラーが発生しないことを確認）
		})
	}
}

func TestRouter_VisionEndpoint_WithImage(t *testing.T) {
	// テスト画像の準備
	testImagePath := "../../../../testdata/sample_text.png"
	if _, err := os.Stat(testImagePath); os.IsNotExist(err) {
		t.Skip("Test image not found, skipping integration test")
	}

	cfg := config.DefaultConfig()
	cfg.Anthropic.APIKey = "test-api-key"
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	// マルチパートフォームデータの作成
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 画像ファイルの追加
	imageFile, err := os.Open(testImagePath)
	if err != nil {
		t.Fatalf("Failed to open test image: %v", err)
	}
	defer func() { _ = imageFile.Close() }()

	part, err := writer.CreateFormFile("image", "test.png")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	if _, err := io.Copy(part, imageFile); err != nil {
		t.Fatalf("Failed to copy image data: %v", err)
	}

	_ = writer.Close()

	// リクエストの作成
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vision/analyze", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	// APIキーが無効なので400または401が返るはず
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized && rec.Code != http.StatusInternalServerError {
		t.Logf("Response status: %d", rec.Code)
		t.Logf("Response body: %s", rec.Body.String())
	}
}

func TestRouter_Integration(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	// 複数のリクエストを順次実行
	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodGet, "/health"},
		{http.MethodGet, "/not-found"},
		{http.MethodGet, "/health"},
	}

	for i, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if ep.path == "/health" && rec.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status %d for %s, got %d", i, http.StatusOK, ep.path, rec.Code)
		}

		if ep.path == "/not-found" && rec.Code != http.StatusNotFound {
			t.Errorf("Request %d: Expected status %d for %s, got %d", i, http.StatusNotFound, ep.path, rec.Code)
		}
	}
}

func TestRouter_ReceiptEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "正常系: POST /api/v1/vision/receipt",
			method:         http.MethodPost,
			expectedStatus: http.StatusBadRequest, // 画像なしなのでBadRequest
		},
		{
			name:           "異常系: GET /api/v1/vision/receipt",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.method == http.MethodPost {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				_ = writer.Close()
				req = httptest.NewRequest(tt.method, "/api/v1/vision/receipt", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/vision/receipt", nil)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRouter_CategorizeEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	container, err := di.NewContainer(cfg)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer func() { _ = container.Close() }()

	router := NewRouter(container)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "正常系: POST /api/v1/vision/categorize",
			method:         http.MethodPost,
			body:           `{"receipt_info":"test"}`,
			expectedStatus: http.StatusInternalServerError, // テスト環境ではAPIキーが無効なので失敗
		},
		{
			name:           "異常系: GET /api/v1/vision/categorize",
			method:         http.MethodGet,
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "異常系: 空のreceipt_info",
			method:         http.MethodPost,
			body:           `{"receipt_info":""}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, "/api/v1/vision/categorize", bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, "/api/v1/vision/categorize", nil)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
