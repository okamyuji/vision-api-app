package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORS(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "正常系: OPTIONS request returns NoContent",
			method:     http.MethodOptions,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "正常系: GET request passes through",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "正常系: POST request passes through",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "正常系: PUT request passes through",
			method:     http.MethodPut,
			wantStatus: http.StatusOK,
		},
		{
			name:       "正常系: DELETE request passes through",
			method:     http.MethodDelete,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.wantStatus)
			}

			// CORSヘッダーの確認
			if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Error("Access-Control-Allow-Origin header not set correctly")
			}

			allowMethods := rec.Header().Get("Access-Control-Allow-Methods")
			if !strings.Contains(allowMethods, "GET") {
				t.Error("Access-Control-Allow-Methods does not contain GET")
			}

			allowHeaders := rec.Header().Get("Access-Control-Allow-Headers")
			if !strings.Contains(allowHeaders, "Content-Type") {
				t.Error("Access-Control-Allow-Headers does not contain Content-Type")
			}

			maxAge := rec.Header().Get("Access-Control-Max-Age")
			if maxAge != "3600" {
				t.Errorf("Access-Control-Max-Age = %s, want 3600", maxAge)
			}
		})
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestLogger(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		responseBody   string
		expectedStatus int
	}{
		{
			name:           "正常系: GET request with 200",
			method:         http.MethodGet,
			path:           "/test",
			statusCode:     http.StatusOK,
			responseBody:   "test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "正常系: POST request with 201",
			method:         http.MethodPost,
			path:           "/api/create",
			statusCode:     http.StatusCreated,
			responseBody:   "created",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "異常系: GET request with 404",
			method:         http.MethodGet,
			path:           "/not-found",
			statusCode:     http.StatusNotFound,
			responseBody:   "not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "異常系: POST request with 500",
			method:         http.MethodPost,
			path:           "/error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "error",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "境界値: Empty response",
			method:         http.MethodGet,
			path:           "/empty",
			statusCode:     http.StatusOK,
			responseBody:   "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if rec.Body.String() != tt.responseBody {
				t.Errorf("response body = %s, want %s", rec.Body.String(), tt.responseBody)
			}
		})
	}
}

func TestLoggerWithHealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		statusCode     int
		expectedStatus int
	}{
		{
			name:           "正常系: /health with 200 (no log)",
			path:           "/health",
			statusCode:     http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "異常系: /health with 500 (log error)",
			path:           "/health",
			statusCode:     http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "正常系: /api/test with 200 (normal log)",
			path:           "/api/test",
			statusCode:     http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "異常系: /api/test with 404 (normal log)",
			path:           "/api/test",
			statusCode:     http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := LoggerWithHealthCheck(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte("test"))
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{
			name:       "正常系: 200 OK",
			statusCode: http.StatusOK,
		},
		{
			name:       "正常系: 201 Created",
			statusCode: http.StatusCreated,
		},
		{
			name:       "異常系: 400 Bad Request",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "異常系: 404 Not Found",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "異常系: 500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: rec,
				statusCode:     http.StatusOK,
			}

			rw.WriteHeader(tt.statusCode)

			if rw.statusCode != tt.statusCode {
				t.Errorf("statusCode = %d, want %d", rw.statusCode, tt.statusCode)
			}

			if rec.Code != tt.statusCode {
				t.Errorf("ResponseWriter.Code = %d, want %d", rec.Code, tt.statusCode)
			}
		})
	}
}

func TestResponseWriter_Write(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantLen int
	}{
		{
			name:    "正常系: Small data",
			data:    []byte("test"),
			wantLen: 4,
		},
		{
			name:    "正常系: Large data",
			data:    []byte(strings.Repeat("a", 1024)),
			wantLen: 1024,
		},
		{
			name:    "境界値: Empty data",
			data:    []byte(""),
			wantLen: 0,
		},
		{
			name:    "境界値: Single byte",
			data:    []byte("a"),
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: rec,
				statusCode:     http.StatusOK,
			}

			n, err := rw.Write(tt.data)
			if err != nil {
				t.Errorf("Write() error = %v", err)
			}

			if n != tt.wantLen {
				t.Errorf("Write() returned %d, want %d", n, tt.wantLen)
			}

			if rw.written != int64(tt.wantLen) {
				t.Errorf("written = %d, want %d", rw.written, tt.wantLen)
			}

			if rec.Body.String() != string(tt.data) {
				t.Errorf("body = %s, want %s", rec.Body.String(), string(tt.data))
			}
		})
	}
}

func TestResponseWriter_MultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// 複数回のWrite
	writes := [][]byte{
		[]byte("Hello"),
		[]byte(" "),
		[]byte("World"),
	}

	var totalWritten int64
	for _, data := range writes {
		n, err := rw.Write(data)
		if err != nil {
			t.Errorf("Write() error = %v", err)
		}
		totalWritten += int64(n)
	}

	if rw.written != totalWritten {
		t.Errorf("written = %d, want %d", rw.written, totalWritten)
	}

	expected := "Hello World"
	if rec.Body.String() != expected {
		t.Errorf("body = %s, want %s", rec.Body.String(), expected)
	}
}

func TestRecovery(t *testing.T) {
	tests := []struct {
		name           string
		panicValue     interface{}
		expectedStatus int
	}{
		{
			name:           "異常系: String panic",
			panicValue:     "test panic",
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "異常系: Error panic",
			panicValue:     http.ErrAbortHandler,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "異常系: Nil panic",
			panicValue:     nil,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "異常系: Integer panic",
			panicValue:     42,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.panicValue)
			}))

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status code = %d, want %d", rec.Code, tt.expectedStatus)
			}

			// レスポンスボディの確認
			var response ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Success {
				t.Error("Expected Success to be false")
			}

			if response.Error != "Internal server error" {
				t.Errorf("Error message = %s, want 'Internal server error'", response.Error)
			}

			// Content-Typeの確認
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", contentType)
			}
		})
	}
}

func TestRecovery_NoPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	if rec.Body.String() != "success" {
		t.Errorf("body = %s, want 'success'", rec.Body.String())
	}
}

func TestMiddlewareChain(t *testing.T) {
	// 複数のミドルウェアを組み合わせたテスト
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	})

	// ミドルウェアチェーン: Recovery -> Logger -> CORS -> Handler
	chain := Recovery(Logger(CORS(handler)))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
	}

	// CORSヘッダーの確認
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header not set in middleware chain")
	}
}

func TestMiddlewareChain_WithPanic(t *testing.T) {
	// panicが発生する場合のミドルウェアチェーン
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic in chain")
	})

	chain := Recovery(Logger(CORS(handler)))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	chain.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	// CORSヘッダーはpanicの前に設定されているはず
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS header not set even before panic")
	}
}
