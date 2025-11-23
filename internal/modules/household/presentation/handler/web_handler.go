package handler

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"

	"vision-api-app/internal/modules/household/usecase"
)

// WebHandler Web UIのハンドラー
type WebHandler struct {
	receiptUseCase   *usecase.ReceiptUseCase
	householdUseCase *usecase.HouseholdUseCase
	templates        map[string]*template.Template
}

// NewWebHandler 新しいWebHandlerを作成
func NewWebHandler(receiptUseCase *usecase.ReceiptUseCase, householdUseCase *usecase.HouseholdUseCase) (*WebHandler, error) {
	// カスタム関数を定義
	funcMap := template.FuncMap{
		"mul": func(a, b int) int {
			return a * b
		},
	}

	// テンプレートを事前にパース
	templates := make(map[string]*template.Template)

	// アップロード画面用テンプレート
	uploadTmpl, err := template.New("base.html").ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "layout", "header.html"),
		filepath.Join("web", "templates", "layout", "footer.html"),
		filepath.Join("web", "templates", "pages", "upload.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse upload template: %w", err)
	}
	templates["upload"] = uploadTmpl

	// 結果表示画面用テンプレート（カスタム関数付き）
	resultTmpl, err := template.New("base.html").Funcs(funcMap).ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "layout", "header.html"),
		filepath.Join("web", "templates", "layout", "footer.html"),
		filepath.Join("web", "templates", "pages", "result.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse result template: %w", err)
	}
	templates["result"] = resultTmpl

	// 家計簿一覧画面用テンプレート
	householdTmpl, err := template.New("base.html").ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "layout", "header.html"),
		filepath.Join("web", "templates", "layout", "footer.html"),
		filepath.Join("web", "templates", "pages", "household.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse household template: %w", err)
	}
	templates["household"] = householdTmpl

	return &WebHandler{
		receiptUseCase:   receiptUseCase,
		householdUseCase: householdUseCase,
		templates:        templates,
	}, nil
}

// HandleUploadPage アップロード画面を表示
func (h *WebHandler) HandleUploadPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data := map[string]interface{}{
		"Title": "レシート登録",
	}

	if err := h.templates["upload"].ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleUpload 画像アップロード処理
func (h *WebHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// マルチパートフォームのパース
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB制限
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// 画像ファイルの取得
	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Image file is required", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// 画像データの読み込み
	imageData, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	// レシート処理
	receipt, err := h.receiptUseCase.ProcessReceiptImage(r.Context(), imageData)
	if err != nil {
		http.Error(w, fmt.Sprintf("レシート認識に失敗しました: %v", err), http.StatusInternalServerError)
		return
	}

	// 結果画面にリダイレクト
	http.Redirect(w, r, fmt.Sprintf("/result?id=%s", receipt.ID), http.StatusSeeOther)
}

// HandleResult 結果表示画面
func (h *WebHandler) HandleResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// IDパラメータの取得
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// レシート取得
	receipt, err := h.receiptUseCase.GetReceipt(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("レシートが見つかりません: %v", err), http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Title":   "レシート詳細",
		"Receipt": receipt,
	}

	if err := h.templates["result"].ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
		return
	}
}

// HandleHousehold 家計簿一覧画面
func (h *WebHandler) HandleHousehold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// レシート一覧を取得
	receipts, err := h.receiptUseCase.ListReceipts(r.Context(), 100, 0)
	if err != nil {
		http.Error(w, "Failed to get receipts", http.StatusInternalServerError)
		return
	}

	// カテゴリ別集計を取得
	summary, err := h.householdUseCase.GetCategorySummary(r.Context())
	if err != nil {
		http.Error(w, "Failed to get category summary", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":           "家計簿一覧",
		"Receipts":        receipts,
		"CategorySummary": summary,
	}

	if err := h.templates["household"].ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
		return
	}
}
