//go:build !no_ai

package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vision-api-app/internal/config"
	"vision-api-app/internal/modules/vision/domain"
)

const (
	// systemPromptReceipt レシート読み取り専用プロンプト
	systemPromptReceipt = `あなたはレシート画像から家計簿用の情報を抽出する専門家です。
JSON形式で正確に情報を返してください。

【レシートの典型的な構造】：
1. 店舗名
2. 商品リスト（商品名と価格）
3. 小計または合計
4. 消費税額
5. お買上金額（これが実際の支払額）
6. お預かり（顧客が渡した金額）← これは支払額ではない！
7. お釣り

【最重要】total_amountの決定ルール：
✅ 正しい：「お買上金額」「合計金額」「小計」
❌ 間違い：「お預かり」「お釣り」「現金」

具体例：
- 商品A: 130円
- 商品B: 529円
- 商品C: 471円
- お買上金額: 1,130円 ← これがtotal_amount
- お預かり: 2,000円 ← これは使わない
- お釣り: 870円 ← これは使わない

【最重要】total_amount の決定方法（この順序で実行）：
1. items リストの price をすべて合計する
2. その合計値を total_amount として使用する
3. レシートに「お買上金額」の表示があっても、items の合計を優先する
4. 「お預かり」「お釣り」は絶対に使用しない

重要：total_amount = sum(items[].price) を必ず守ってください。

【商品リストの作成】：
実際に購入した商品のみを items に含める。
以下は商品ではないので絶対に除外：
- 「お預かり」
- 「お釣り」
- 「(内)消費税額」
- 「点数」
- 「現金」
- 「合計」
- 「小計」

必須項目：
- store_name: 店舗名
- purchase_date: 購入日時（YYYY-MM-DD HH:MM形式、時刻不明なら12:00）
- total_amount: お買上金額（商品の合計金額、必ずitemsの合計と一致）
- tax_amount: 消費税額（不明な場合は0）
- items: 商品リスト（name, quantity, price）

オプション項目：
- payment_method: 支払い方法
- receipt_number: レシート番号

出力形式：
{
  "store_name": "店舗名",
  "purchase_date": "2025-11-22 14:30",
  "total_amount": 1500,
  "tax_amount": 150,
  "payment_method": "現金",
  "items": [
    {"name": "商品名", "quantity": 1, "price": 500}
  ]
}

注意：
- 金額は数値型（カンマや円記号を除く）
- total_amount は必ず items の price の合計と一致させる
- JSONのみを返す（説明不要）`

	// systemPromptCategorize 仕訳け専用プロンプト
	systemPromptCategorize = `あなたは家計簿の仕訳け専門家です。
レシート情報から適切なカテゴリを判定してください。

利用可能なカテゴリ：
- 食費: 食品、飲料、外食
- 日用品: 洗剤、ティッシュ、トイレットペーパー等
- 交通費: 電車、バス、タクシー、ガソリン
- 医療費: 病院、薬局、薬
- 娯楽費: 映画、書籍、ゲーム、趣味
- 衣服費: 衣類、靴、アクセサリー
- 通信費: 携帯電話、インターネット
- 光熱費: 電気、ガス、水道
- 教育費: 学費、教材、習い事
- その他: 上記に該当しないもの

入力されたレシート情報から、最も適切なカテゴリを1つ選択してください。

出力形式：
{
  "category": "カテゴリ名",
  "confidence": 0.95,
  "reason": "判定理由（簡潔に）"
}

判定基準：
1. 店舗名から判断（例：スーパー→食費、ドラッグストア→日用品または医療費）
2. 商品名から判断（複数カテゴリにまたがる場合は主要な商品で判定）
3. 金額や購入パターンも考慮
4. 確信度（confidence）は0.0〜1.0で返す
5. JSONのみを返す（説明文は不要）`

	// systemPromptGeneral 汎用テキスト抽出プロンプト
	systemPromptGeneral = `この画像に含まれるすべてのテキストを正確に抽出してください。

抽出ルール：
1. 画像内のすべてのテキストを漏れなく抽出する
2. レイアウトや改行を可能な限り保持する
3. 日本語と英語の両方に対応する
4. 数字、記号も正確に抽出する
5. 読み取れない文字は[?]で表記する
6. 抽出したテキストのみを返す（説明不要）

出力形式：
抽出したテキストをそのまま返してください。`
)

// ClaudeRepository Claude APIのリポジトリ実装
type ClaudeRepository struct {
	apiKey      string
	model       string
	maxTokens   int
	httpClient  *http.Client
	apiEndpoint string // テスト用にエンドポイントを差し替え可能に
}

// NewClaudeRepository 新しいClaudeRepositoryを作成
func NewClaudeRepository(cfg *config.AnthropicConfig) *ClaudeRepository {
	return &ClaudeRepository{
		apiKey:      cfg.APIKey,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		apiEndpoint: "https://api.anthropic.com/v1/messages",
	}
}

// SetHTTPClient テスト用にHTTPクライアントを設定（テストコードからのみ使用）
func (r *ClaudeRepository) SetHTTPClient(client *http.Client) {
	r.httpClient = client
}

// Correct テキストを補正（汎用）
func (r *ClaudeRepository) Correct(text string) (*domain.AIResult, error) {
	requestBody := map[string]interface{}{
		"model":      r.model,
		"max_tokens": r.maxTokens,
		"system":     systemPromptGeneral,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": text},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", r.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", r.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	correctedText := text
	if len(response.Content) > 0 {
		correctedText = response.Content[0].Text
	}

	return domain.NewAIResult(
		text,
		correctedText,
		response.Usage.InputTokens,
		response.Usage.OutputTokens,
		r.model,
	), nil
}

// RecognizeImage 画像から直接テキストを認識（汎用）
func (r *ClaudeRepository) RecognizeImage(imageData []byte) (*domain.AIResult, error) {
	return r.recognizeImageWithPrompt(imageData, systemPromptGeneral, "この画像からすべてのテキストを抽出してください。")
}

// RecognizeReceipt レシート画像から構造化データを抽出
func (r *ClaudeRepository) RecognizeReceipt(imageData []byte) (*domain.AIResult, error) {
	return r.recognizeImageWithPrompt(imageData, systemPromptReceipt, "このレシート画像から情報を抽出してJSON形式で返してください。")
}

// CategorizeReceipt レシート情報から適切なカテゴリを判定
func (r *ClaudeRepository) CategorizeReceipt(receiptInfo string) (*domain.AIResult, error) {
	requestBody := map[string]interface{}{
		"model":      r.model,
		"max_tokens": r.maxTokens,
		"system":     systemPromptCategorize,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]string{
					{"type": "text", "text": receiptInfo},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", r.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", r.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	categorizedText := ""
	if len(response.Content) > 0 {
		categorizedText = response.Content[0].Text
	}

	return domain.NewAIResult(
		receiptInfo,
		categorizedText,
		response.Usage.InputTokens,
		response.Usage.OutputTokens,
		r.model,
	), nil
}

// recognizeImageWithPrompt 画像認識の共通処理
func (r *ClaudeRepository) recognizeImageWithPrompt(imageData []byte, systemPrompt, userPrompt string) (*domain.AIResult, error) {
	// 画像をbase64エンコード
	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	// 画像の形式を判定（簡易版）
	mediaType := "image/png"
	if len(imageData) > 2 && imageData[0] == 0xFF && imageData[1] == 0xD8 {
		mediaType = "image/jpeg"
	}

	requestBody := map[string]interface{}{
		"model":      r.model,
		"max_tokens": r.maxTokens,
		"system":     systemPrompt,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": mediaType,
							"data":       imageBase64,
						},
					},
					{
						"type": "text",
						"text": userPrompt,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", r.apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", r.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	recognizedText := ""
	if len(response.Content) > 0 {
		recognizedText = response.Content[0].Text
	}

	return domain.NewAIResult(
		"",
		recognizedText,
		response.Usage.InputTokens,
		response.Usage.OutputTokens,
		r.model,
	), nil
}

// ProviderName プロバイダー名を返す
func (r *ClaudeRepository) ProviderName() string {
	return "Anthropic Claude"
}
