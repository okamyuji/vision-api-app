package entity

import (
	"testing"
	"time"
)

func TestNewReceipt(t *testing.T) {
	id := "test-id"
	storeName := "テストストア"
	purchaseDate := time.Now()
	totalAmount := 1000
	taxAmount := 100
	category := "食費"

	receipt := NewReceipt(id, storeName, purchaseDate, totalAmount, taxAmount, category)

	if receipt.ID != id {
		t.Errorf("ID = %v, want %v", receipt.ID, id)
	}
	if receipt.StoreName != storeName {
		t.Errorf("StoreName = %v, want %v", receipt.StoreName, storeName)
	}
	if receipt.TotalAmount != totalAmount {
		t.Errorf("TotalAmount = %v, want %v", receipt.TotalAmount, totalAmount)
	}
	if receipt.TaxAmount != taxAmount {
		t.Errorf("TaxAmount = %v, want %v", receipt.TaxAmount, taxAmount)
	}
	if receipt.Category != category {
		t.Errorf("Category = %v, want %v", receipt.Category, category)
	}
	if len(receipt.Items) != 0 {
		t.Errorf("Items length = %v, want 0", len(receipt.Items))
	}
}

func TestReceipt_AddItem(t *testing.T) {
	receipt := NewReceipt("receipt-id", "ストア", time.Now(), 1000, 100, "食費")

	item1 := NewReceiptItem("item-1", receipt.ID, "商品1", 2, 500)
	item2 := NewReceiptItem("item-2", receipt.ID, "商品2", 1, 300)

	receipt.AddItem(item1)
	if receipt.TotalItems() != 1 {
		t.Errorf("TotalItems() = %v, want 1", receipt.TotalItems())
	}

	receipt.AddItem(item2)
	if receipt.TotalItems() != 2 {
		t.Errorf("TotalItems() = %v, want 2", receipt.TotalItems())
	}
}

func TestReceipt_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		storeName   string
		totalAmount int
		want        bool
	}{
		{"正常_通常のレシート", "ストア", 1000, true},
		{"正常_ゼロ金額", "ストア", 0, true},
		{"異常_空のストア名", "", 1000, false},
		{"異常_負の金額", "ストア", -100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := NewReceipt(
				"test-id",
				tt.storeName,
				time.Now(),
				tt.totalAmount,
				0,
				"",
			)

			if got := receipt.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewReceiptItem(t *testing.T) {
	id := "item-id"
	receiptID := "receipt-id"
	name := "商品名"
	quantity := 3
	price := 500

	item := NewReceiptItem(id, receiptID, name, quantity, price)

	if item.ID != id {
		t.Errorf("ID = %v, want %v", item.ID, id)
	}
	if item.ReceiptID != receiptID {
		t.Errorf("ReceiptID = %v, want %v", item.ReceiptID, receiptID)
	}
	if item.Name != name {
		t.Errorf("Name = %v, want %v", item.Name, name)
	}
	if item.Quantity != quantity {
		t.Errorf("Quantity = %v, want %v", item.Quantity, quantity)
	}
	if item.Price != price {
		t.Errorf("Price = %v, want %v", item.Price, price)
	}
}

func TestReceiptItem_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		itemName string
		quantity int
		price    int
		want     bool
	}{
		{"正常_通常の商品", "商品", 1, 100, true},
		{"正常_複数個", "商品", 5, 100, true},
		{"正常_ゼロ円", "商品", 1, 0, true},
		{"異常_空の商品名", "", 1, 100, false},
		{"異常_ゼロ数量", "商品", 0, 100, false},
		{"異常_負の数量", "商品", -1, 100, false},
		{"異常_負の価格", "商品", 1, -100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := NewReceiptItem(
				"item-id",
				"receipt-id",
				tt.itemName,
				tt.quantity,
				tt.price,
			)

			if got := item.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewExpenseEntry(t *testing.T) {
	id := "entry-id"
	date := time.Now()
	category := "食費"
	amount := 1500
	description := "ランチ"
	tags := []string{"外食", "平日"}

	entry := NewExpenseEntry(id, date, category, amount, description, tags)

	if entry.ID != id {
		t.Errorf("ID = %v, want %v", entry.ID, id)
	}
	if entry.Category != category {
		t.Errorf("Category = %v, want %v", entry.Category, category)
	}
	if entry.Amount != amount {
		t.Errorf("Amount = %v, want %v", entry.Amount, amount)
	}
	if entry.Description != description {
		t.Errorf("Description = %v, want %v", entry.Description, description)
	}
	if len(entry.Tags) != 2 {
		t.Errorf("Tags length = %v, want 2", len(entry.Tags))
	}
}

func TestExpenseEntry_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		category string
		amount   int
		want     bool
	}{
		{"正常_通常のエントリ", "食費", 1000, true},
		{"正常_ゼロ金額", "食費", 0, true},
		{"異常_空のカテゴリ", "", 1000, false},
		{"異常_負の金額", "食費", -100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := NewExpenseEntry(
				"entry-id",
				time.Now(),
				tt.category,
				tt.amount,
				"",
				[]string{},
			)

			if got := entry.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCategory(t *testing.T) {
	id := "category-id"
	name := "食費"
	description := "食料品・飲料"
	color := "#FF6B6B"

	category := NewCategory(id, name, description, color)

	if category.ID != id {
		t.Errorf("ID = %v, want %v", category.ID, id)
	}
	if category.Name != name {
		t.Errorf("Name = %v, want %v", category.Name, name)
	}
	if category.Description != description {
		t.Errorf("Description = %v, want %v", category.Description, description)
	}
	if category.Color != color {
		t.Errorf("Color = %v, want %v", category.Color, color)
	}
}

func TestCategory_IsValid(t *testing.T) {
	tests := []struct {
		name         string
		categoryName string
		want         bool
	}{
		{"正常_通常のカテゴリ", "食費", true},
		{"異常_空の名前", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category := NewCategory(
				"category-id",
				tt.categoryName,
				"",
				"",
			)

			if got := category.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReceipt_TotalItems(t *testing.T) {
	receipt := NewReceipt("receipt-id", "ストア", time.Now(), 1000, 100, "食費")

	// 初期状態
	if receipt.TotalItems() != 0 {
		t.Errorf("TotalItems() = %v, want 0", receipt.TotalItems())
	}

	// 3個追加
	for i := 0; i < 3; i++ {
		item := NewReceiptItem("item-id", receipt.ID, "商品", 1, 100)
		receipt.AddItem(item)
	}

	if receipt.TotalItems() != 3 {
		t.Errorf("TotalItems() = %v, want 3", receipt.TotalItems())
	}
}
