package domain

import (
	"testing"
	"time"
)

func TestNewAIResult(t *testing.T) {
	originalText := "original"
	correctedText := "corrected"
	inputTokens := 100
	outputTokens := 50
	model := "claude-haiku-4-5"

	result := NewAIResult(originalText, correctedText, inputTokens, outputTokens, model)

	if result.OriginalText != originalText {
		t.Errorf("Expected OriginalText %s, got %s", originalText, result.OriginalText)
	}
	if result.CorrectedText != correctedText {
		t.Errorf("Expected CorrectedText %s, got %s", correctedText, result.CorrectedText)
	}
	if result.InputTokens != inputTokens {
		t.Errorf("Expected InputTokens %d, got %d", inputTokens, result.InputTokens)
	}
	if result.OutputTokens != outputTokens {
		t.Errorf("Expected OutputTokens %d, got %d", outputTokens, result.OutputTokens)
	}
	if result.Model != model {
		t.Errorf("Expected Model %s, got %s", model, result.Model)
	}
	if result.ProcessedAt.IsZero() {
		t.Error("Expected ProcessedAt to be set")
	}
	if time.Since(result.ProcessedAt) > time.Second {
		t.Error("Expected ProcessedAt to be recent")
	}
}

func TestAIResult_IsCorrected(t *testing.T) {
	tests := []struct {
		name          string
		originalText  string
		correctedText string
		want          bool
	}{
		{
			name:          "テキストが補正された場合",
			originalText:  "original",
			correctedText: "corrected",
			want:          true,
		},
		{
			name:          "テキストが補正されていない場合",
			originalText:  "same",
			correctedText: "same",
			want:          false,
		},
		{
			name:          "空文字列の場合",
			originalText:  "",
			correctedText: "",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAIResult(tt.originalText, tt.correctedText, 0, 0, "test")
			if got := result.IsCorrected(); got != tt.want {
				t.Errorf("IsCorrected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIResult_TotalTokens(t *testing.T) {
	tests := []struct {
		name         string
		inputTokens  int
		outputTokens int
		want         int
	}{
		{
			name:         "通常のトークン数",
			inputTokens:  100,
			outputTokens: 50,
			want:         150,
		},
		{
			name:         "ゼロトークン",
			inputTokens:  0,
			outputTokens: 0,
			want:         0,
		},
		{
			name:         "大きなトークン数",
			inputTokens:  10000,
			outputTokens: 5000,
			want:         15000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAIResult("", "", tt.inputTokens, tt.outputTokens, "test")
			if got := result.TotalTokens(); got != tt.want {
				t.Errorf("TotalTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}
