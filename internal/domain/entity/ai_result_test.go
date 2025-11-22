package entity

import "testing"

func TestNewAIResult(t *testing.T) {
	original := "original text"
	corrected := "corrected text"
	inputTokens := 10
	outputTokens := 15
	model := "claude-3-5-sonnet"

	result := NewAIResult(original, corrected, inputTokens, outputTokens, model)

	if result.OriginalText != original {
		t.Errorf("Expected original text %s, got %s", original, result.OriginalText)
	}
	if result.CorrectedText != corrected {
		t.Errorf("Expected corrected text %s, got %s", corrected, result.CorrectedText)
	}
	if result.InputTokens != inputTokens {
		t.Errorf("Expected input tokens %d, got %d", inputTokens, result.InputTokens)
	}
	if result.OutputTokens != outputTokens {
		t.Errorf("Expected output tokens %d, got %d", outputTokens, result.OutputTokens)
	}
	if result.Model != model {
		t.Errorf("Expected model %s, got %s", model, result.Model)
	}
}

func TestAIResult_IsCorrected(t *testing.T) {
	tests := []struct {
		name      string
		original  string
		corrected string
		want      bool
	}{
		{"補正あり", "original", "corrected", true},
		{"補正なし", "same", "same", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewAIResult(tt.original, tt.corrected, 10, 10, "model")
			if got := result.IsCorrected(); got != tt.want {
				t.Errorf("IsCorrected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIResult_TotalTokens(t *testing.T) {
	result := NewAIResult("text", "text", 10, 15, "model")
	expected := 25

	if got := result.TotalTokens(); got != expected {
		t.Errorf("TotalTokens() = %d, want %d", got, expected)
	}
}
