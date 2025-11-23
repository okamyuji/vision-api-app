package domain

import "time"

// AIResult AI補正結果のエンティティ
type AIResult struct {
	OriginalText  string
	CorrectedText string
	InputTokens   int
	OutputTokens  int
	Model         string
	ProcessedAt   time.Time
}

// NewAIResult 新しいAIResultを作成
func NewAIResult(originalText, correctedText string, inputTokens, outputTokens int, model string) *AIResult {
	return &AIResult{
		OriginalText:  originalText,
		CorrectedText: correctedText,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		Model:         model,
		ProcessedAt:   time.Now(),
	}
}

// IsCorrected テキストが補正されたかどうかを判定
func (r *AIResult) IsCorrected() bool {
	return r.OriginalText != r.CorrectedText
}

// TotalTokens 合計トークン数を返す
func (r *AIResult) TotalTokens() int {
	return r.InputTokens + r.OutputTokens
}
