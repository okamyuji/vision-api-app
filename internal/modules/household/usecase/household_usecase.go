package usecase

import (
	"context"

	"vision-api-app/internal/modules/household/domain/repository"
)

// CategorySummary カテゴリ別集計結果
type CategorySummary struct {
	Category string
	Count    int
	Total    int
}

// HouseholdUseCase 家計簿集計のユースケース
type HouseholdUseCase struct {
	receiptRepo repository.ReceiptRepository
	expenseRepo repository.ExpenseRepository
}

// NewHouseholdUseCase 新しいHouseholdUseCaseを作成
func NewHouseholdUseCase(receiptRepo repository.ReceiptRepository, expenseRepo repository.ExpenseRepository) *HouseholdUseCase {
	return &HouseholdUseCase{
		receiptRepo: receiptRepo,
		expenseRepo: expenseRepo,
	}
}

// GetCategorySummary カテゴリ別集計を取得（明細項目ベース + expense_entries）
func (uc *HouseholdUseCase) GetCategorySummary(ctx context.Context) ([]CategorySummary, error) {
	// レシート一覧を取得
	receipts, err := uc.receiptRepo.FindAll(ctx, 0, 0)
	if err != nil {
		return nil, err
	}

	// 家計簿エントリ一覧を取得
	expenses, err := uc.expenseRepo.FindAll(ctx, 0, 0)
	if err != nil {
		return nil, err
	}

	// カテゴリ別に集計
	summaryMap := make(map[string]*CategorySummary)

	// レシートの明細項目を集計（項目ごとに仕訳）
	for _, receipt := range receipts {
		for _, item := range receipt.Items {
			category := item.Category
			if category == "" {
				category = "その他"
			}
			if _, exists := summaryMap[category]; !exists {
				summaryMap[category] = &CategorySummary{
					Category: category,
					Count:    0,
					Total:    0,
				}
			}
			summaryMap[category].Count++
			summaryMap[category].Total += item.Price * item.Quantity
		}
	}

	// 家計簿エントリを集計
	for _, expense := range expenses {
		if expense.Category == "" {
			continue
		}
		if _, exists := summaryMap[expense.Category]; !exists {
			summaryMap[expense.Category] = &CategorySummary{
				Category: expense.Category,
				Count:    0,
				Total:    0,
			}
		}
		summaryMap[expense.Category].Count++
		summaryMap[expense.Category].Total += expense.Amount
	}

	// マップをスライスに変換
	summaries := make([]CategorySummary, 0, len(summaryMap))
	for _, summary := range summaryMap {
		summaries = append(summaries, *summary)
	}

	return summaries, nil
}
