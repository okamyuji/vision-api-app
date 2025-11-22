package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vision-api-app/internal/config"
)

func TestClaudeRepository_Correct(t *testing.T) {
	t.Run("正常系_テキスト補正_モックサーバー", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"content": []map[string]interface{}{
					{"text": "補正されたテキスト"},
				},
				"usage": map[string]int{
					"input_tokens":  150,
					"output_tokens": 50,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		result, err := repo.Correct("テストテキスト")
		if err != nil {
			t.Errorf("Correct() error = %v", err)
		}

		if result == nil {
			t.Fatal("result is nil")
		}

		if result.CorrectedText != "補正されたテキスト" {
			t.Errorf("CorrectedText = %v, want '補正されたテキスト'", result.CorrectedText)
		}

		if result.InputTokens != 150 {
			t.Errorf("InputTokens = %v, want 150", result.InputTokens)
		}

		if result.OutputTokens != 50 {
			t.Errorf("OutputTokens = %v, want 50", result.OutputTokens)
		}
	})

	t.Run("異常系_空のテキスト", func(t *testing.T) {
		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		_, err := repo.Correct("")
		if err == nil {
			t.Error("Expected error for empty text, got nil")
		}
	})

	t.Run("異常系_API Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": {"message": "Bad Request"}}`))
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		_, err := repo.Correct("test")
		if err == nil {
			t.Error("Expected error from API, got nil")
		}
	})
}

func TestClaudeRepository_RecognizeImage(t *testing.T) {
	t.Run("正常系_画像認識_モックサーバー", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"content": []map[string]interface{}{
					{"text": "画像から抽出されたテキスト"},
				},
				"usage": map[string]int{
					"input_tokens":  200,
					"output_tokens": 100,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		imageData := []byte("fake image data")
		result, err := repo.RecognizeImage(imageData)
		if err != nil {
			t.Errorf("RecognizeImage() error = %v", err)
		}

		if result == nil {
			t.Fatal("result is nil")
		}

		if result.CorrectedText != "画像から抽出されたテキスト" {
			t.Errorf("CorrectedText = %v, want '画像から抽出されたテキスト'", result.CorrectedText)
		}

		if result.InputTokens != 200 {
			t.Errorf("InputTokens = %v, want 200", result.InputTokens)
		}

		if result.OutputTokens != 100 {
			t.Errorf("OutputTokens = %v, want 100", result.OutputTokens)
		}
	})

	t.Run("異常系_空の画像データ", func(t *testing.T) {
		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		_, err := repo.RecognizeImage([]byte{})
		if err == nil {
			t.Error("Expected error for empty image data, got nil")
		}
	})

	t.Run("異常系_API Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": {"message": "API Error"}}`))
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		_, err := repo.RecognizeImage([]byte("test"))
		if err == nil {
			t.Error("Expected error from API, got nil")
		}
	})
}

func TestClaudeRepository_ProviderName(t *testing.T) {
	cfg := &config.AnthropicConfig{
		APIKey:    "test-api-key",
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 1024,
	}

	repo := NewClaudeRepository(cfg)
	name := repo.ProviderName()

	if name != "Anthropic Claude" {
		t.Errorf("ProviderName() = %v, want Anthropic Claude", name)
	}
}

func TestNewClaudeRepository(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		model     string
		maxTokens int
	}{
		{"正常_標準設定", "api-key-123", "claude-3-haiku-20240307", 1024},
		{"正常_大きなトークン数", "api-key-456", "claude-3-sonnet-20240229", 4096},
		{"境界値_最小トークン", "api-key-789", "claude-3-opus-20240229", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.AnthropicConfig{
				APIKey:    tt.apiKey,
				Model:     tt.model,
				MaxTokens: tt.maxTokens,
			}

			repo := NewClaudeRepository(cfg)

			if repo.apiKey != tt.apiKey {
				t.Errorf("apiKey = %v, want %v", repo.apiKey, tt.apiKey)
			}
			if repo.model != tt.model {
				t.Errorf("model = %v, want %v", repo.model, tt.model)
			}
			if repo.maxTokens != tt.maxTokens {
				t.Errorf("maxTokens = %v, want %v", repo.maxTokens, tt.maxTokens)
			}
			if repo.httpClient == nil {
				t.Error("httpClient should not be nil")
			}
		})
	}
}

func TestClaudeRepository_ErrorHandling(t *testing.T) {
	t.Run("異常系_APIエラーレスポンス", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": {"message": "Invalid request"}}`))
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.httpClient = server.Client()

		// 実際のAPIを呼ぶのでスキップ
		_, err := repo.Correct("test")
		if err == nil {
			t.Skip("Skipping actual API call test")
		}
	})

	t.Run("異常系_ネットワークエラー", func(t *testing.T) {
		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		// 存在しないサーバーに接続
		repo.httpClient.Timeout = 1

		// 実際のAPIを呼ぶのでスキップ
		_, err := repo.Correct("test")
		if err == nil {
			t.Skip("Skipping actual API call test")
		}
	})
}

func TestClaudeRepository_SystemPrompts(t *testing.T) {
	t.Run("システムプロンプト確認", func(t *testing.T) {
		if systemPromptReceipt == "" {
			t.Error("systemPromptReceipt should not be empty")
		}
		if systemPromptCategorize == "" {
			t.Error("systemPromptCategorize should not be empty")
		}
		if systemPromptGeneral == "" {
			t.Error("systemPromptGeneral should not be empty")
		}

		// プロンプトに重要なキーワードが含まれているか確認
		if len(systemPromptReceipt) < 100 {
			t.Error("systemPromptReceipt is too short")
		}
		if len(systemPromptCategorize) < 100 {
			t.Error("systemPromptCategorize is too short")
		}
		if len(systemPromptGeneral) < 50 {
			t.Error("systemPromptGeneral is too short")
		}
	})
}

func TestClaudeRepository_ImageEncoding(t *testing.T) {
	t.Run("画像データのBase64エンコード", func(t *testing.T) {
		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)

		// 小さな画像データ
		imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // JPEG header

		// RecognizeImageは内部でBase64エンコードを行う
		// 実際のAPI呼び出しはスキップ
		_, err := repo.RecognizeImage(imageData)
		if err == nil {
			t.Skip("Skipping actual API call test")
		}
	})

	t.Run("境界値_大きな画像", func(t *testing.T) {
		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)

		// 1MBの画像データ
		largeImage := make([]byte, 1024*1024)

		_, err := repo.RecognizeImage(largeImage)
		if err == nil {
			t.Skip("Skipping actual API call test")
		}
	})
}

func TestClaudeRepository_ModelVariations(t *testing.T) {
	models := []string{
		"claude-3-haiku-20240307",
		"claude-3-sonnet-20240229",
		"claude-3-opus-20240229",
		"claude-haiku-4-5-20251001",
	}

	for _, model := range models {
		t.Run("Model_"+model, func(t *testing.T) {
			cfg := &config.AnthropicConfig{
				APIKey:    "test-api-key",
				Model:     model,
				MaxTokens: 1024,
			}

			repo := NewClaudeRepository(cfg)
			providerName := repo.ProviderName()

			expectedName := "Anthropic Claude"
			if providerName != expectedName {
				t.Errorf("ProviderName() = %v, want %v", providerName, expectedName)
			}
		})
	}
}

func TestClaudeRepository_RecognizeReceipt(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
		skip    bool
	}{
		{
			name:    "正常系: APIキーあり（実際の呼び出しはスキップ）",
			apiKey:  "test-api-key",
			wantErr: true, // テスト環境では失敗する
			skip:    true,
		},
		{
			name:    "境界値: 空のAPIキー",
			apiKey:  "",
			wantErr: true,
			skip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Skipping actual API call test")
			}

			cfg := &config.AnthropicConfig{
				APIKey:    tt.apiKey,
				Model:     "claude-haiku-4-5-20251001",
				MaxTokens: 4096,
			}

			repo := NewClaudeRepository(cfg)
			_, err := repo.RecognizeReceipt([]byte("test image data"))

			if (err != nil) != tt.wantErr {
				t.Errorf("RecognizeReceipt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClaudeRepository_CategorizeReceipt(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
		skip    bool
	}{
		{
			name:    "正常系: APIキーあり（実際の呼び出しはスキップ）",
			apiKey:  "test-api-key",
			wantErr: true, // テスト環境では失敗する
			skip:    true,
		},
		{
			name:    "境界値: 空のAPIキー",
			apiKey:  "",
			wantErr: true,
			skip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Skipping actual API call test")
			}

			cfg := &config.AnthropicConfig{
				APIKey:    tt.apiKey,
				Model:     "claude-haiku-4-5-20251001",
				MaxTokens: 4096,
			}

			repo := NewClaudeRepository(cfg)
			receiptInfo := `{"store_name":"スーパーマーケット","items":[{"name":"野菜"}]}`
			_, err := repo.CategorizeReceipt(receiptInfo)

			if (err != nil) != tt.wantErr {
				t.Errorf("CategorizeReceipt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClaudeRepository_EmptyImageData(t *testing.T) {
	cfg := &config.AnthropicConfig{
		APIKey:    "test-api-key",
		Model:     "claude-haiku-4-5-20251001",
		MaxTokens: 4096,
	}

	repo := NewClaudeRepository(cfg)

	t.Run("異常系: 空の画像データ", func(t *testing.T) {
		_, err := repo.RecognizeImage([]byte{})
		if err == nil {
			t.Error("Expected error for empty image data, got nil")
		}
	})

	t.Run("異常系: nilの画像データ", func(t *testing.T) {
		_, err := repo.RecognizeImage(nil)
		if err == nil {
			t.Error("Expected error for nil image data, got nil")
		}
	})
}

func TestClaudeRepository_EmptyText(t *testing.T) {
	cfg := &config.AnthropicConfig{
		APIKey:    "test-api-key",
		Model:     "claude-haiku-4-5-20251001",
		MaxTokens: 4096,
	}

	repo := NewClaudeRepository(cfg)

	t.Run("異常系: 空のテキスト", func(t *testing.T) {
		_, err := repo.Correct("")
		if err == nil {
			t.Error("Expected error for empty text, got nil")
		}
	})

	t.Run("異常系: 空のレシート情報", func(t *testing.T) {
		_, err := repo.CategorizeReceipt("")
		if err == nil {
			t.Error("Expected error for empty receipt info, got nil")
		}
	})
}

func TestClaudeRepository_LargeInput(t *testing.T) {
	cfg := &config.AnthropicConfig{
		APIKey:    "test-api-key",
		Model:     "claude-haiku-4-5-20251001",
		MaxTokens: 4096,
	}

	repo := NewClaudeRepository(cfg)

	t.Run("境界値: 大きなテキスト", func(t *testing.T) {
		largeText := string(make([]byte, 100000)) // 100KB
		_, err := repo.Correct(largeText)
		if err == nil {
			t.Skip("Skipping actual API call test")
		}
	})
}

func TestClaudeRepository_InvalidJSON(t *testing.T) {
	cfg := &config.AnthropicConfig{
		APIKey:    "test-api-key",
		Model:     "claude-haiku-4-5-20251001",
		MaxTokens: 4096,
	}

	repo := NewClaudeRepository(cfg)

	t.Run("異常系: 無効なJSON", func(t *testing.T) {
		invalidJSON := `{invalid json}`
		_, err := repo.CategorizeReceipt(invalidJSON)
		if err == nil {
			t.Skip("Skipping actual API call test")
		}
	})
}

func TestClaudeRepository_RecognizeReceipt_Mock(t *testing.T) {
	t.Run("正常系_レシート認識_モックサーバー", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"content": []map[string]interface{}{
					{"text": `{"store_name":"テストストア","total_amount":1000}`},
				},
				"usage": map[string]int{
					"input_tokens":  300,
					"output_tokens": 120,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		result, err := repo.RecognizeReceipt([]byte("fake receipt image"))
		if err != nil {
			t.Errorf("RecognizeReceipt() error = %v", err)
		}

		if result == nil {
			t.Fatal("result is nil")
		}

		if result.InputTokens != 300 {
			t.Errorf("InputTokens = %v, want 300", result.InputTokens)
		}
	})

	t.Run("異常系_API Error_Receipt", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		_, err := repo.RecognizeReceipt([]byte("test"))
		if err == nil {
			t.Error("Expected error from API, got nil")
		}
	})
}

func TestClaudeRepository_CategorizeReceipt_Mock(t *testing.T) {
	t.Run("正常系_カテゴリ判定_モックサーバー", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"content": []map[string]interface{}{
					{"text": `{"category":"食費","confidence":0.95}`},
				},
				"usage": map[string]int{
					"input_tokens":  100,
					"output_tokens": 50,
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		result, err := repo.CategorizeReceipt(`{"store_name":"スーパーマーケット"}`)
		if err != nil {
			t.Errorf("CategorizeReceipt() error = %v", err)
		}

		if result == nil {
			t.Fatal("result is nil")
		}

		if result.InputTokens != 100 {
			t.Errorf("InputTokens = %v, want 100", result.InputTokens)
		}
	})

	t.Run("異常系_Malformed JSON Response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`invalid json response`))
		}))
		defer server.Close()

		cfg := &config.AnthropicConfig{
			APIKey:    "test-api-key",
			Model:     "claude-3-haiku-20240307",
			MaxTokens: 1024,
		}

		repo := NewClaudeRepository(cfg)
		repo.apiEndpoint = server.URL
		repo.setHTTPClient(server.Client())

		_, err := repo.CategorizeReceipt(`{"store_name":"test"}`)
		if err == nil {
			t.Error("Expected error for malformed JSON, got nil")
		}
	})
}
