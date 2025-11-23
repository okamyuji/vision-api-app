package entity

import (
	"time"
)

// Receipt レシートエンティティ
type Receipt struct {
	ID            string
	StoreName     string
	PurchaseDate  time.Time
	TotalAmount   int    // 実際に使った金額
	TaxAmount     int    // 消費税額
	PaymentMethod string // 支払い方法
	ReceiptNumber string // レシート番号
	Category      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	Items         []ReceiptItem
}

// ReceiptItem レシート明細エンティティ
type ReceiptItem struct {
	ID        string
	ReceiptID string
	Name      string
	Quantity  int
	Price     int
	CreatedAt time.Time
}

// ExpenseEntry 家計簿エントリエンティティ
type ExpenseEntry struct {
	ID          string
	ReceiptID   *string
	Date        time.Time
	Category    string
	Amount      int
	Description string
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Category カテゴリエンティティ
type Category struct {
	ID          string
	Name        string
	Description string
	Color       string
	CreatedAt   time.Time
}

// NewReceipt 新しいReceiptを作成
func NewReceipt(id, storeName string, purchaseDate time.Time, totalAmount, taxAmount int, category string) *Receipt {
	now := time.Now()
	return &Receipt{
		ID:            id,
		StoreName:     storeName,
		PurchaseDate:  purchaseDate,
		TotalAmount:   totalAmount,
		TaxAmount:     taxAmount,
		PaymentMethod: "",
		ReceiptNumber: "",
		Category:      category,
		CreatedAt:     now,
		UpdatedAt:     now,
		Items:         []ReceiptItem{},
	}
}

// NewReceiptItem 新しいReceiptItemを作成
func NewReceiptItem(id, receiptID, name string, quantity, price int) *ReceiptItem {
	return &ReceiptItem{
		ID:        id,
		ReceiptID: receiptID,
		Name:      name,
		Quantity:  quantity,
		Price:     price,
		CreatedAt: time.Now(),
	}
}

// NewExpenseEntry 新しいExpenseEntryを作成
func NewExpenseEntry(id string, date time.Time, category string, amount int, description string, tags []string) *ExpenseEntry {
	now := time.Now()
	return &ExpenseEntry{
		ID:          id,
		Date:        date,
		Category:    category,
		Amount:      amount,
		Description: description,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewCategory 新しいCategoryを作成
func NewCategory(id, name, description, color string) *Category {
	return &Category{
		ID:          id,
		Name:        name,
		Description: description,
		Color:       color,
		CreatedAt:   time.Now(),
	}
}

// AddItem レシートに明細を追加
func (r *Receipt) AddItem(item *ReceiptItem) {
	r.Items = append(r.Items, *item)
}

// TotalItems 明細の合計数を返す
func (r *Receipt) TotalItems() int {
	return len(r.Items)
}

// IsValid レシートが有効かチェック
func (r *Receipt) IsValid() bool {
	return r.StoreName != "" && r.TotalAmount >= 0
}

// IsValid 明細が有効かチェック
func (ri *ReceiptItem) IsValid() bool {
	return ri.Name != "" && ri.Quantity > 0 && ri.Price >= 0
}

// IsValid 家計簿エントリが有効かチェック
func (e *ExpenseEntry) IsValid() bool {
	return e.Category != "" && e.Amount >= 0
}

// IsValid カテゴリが有効かチェック
func (c *Category) IsValid() bool {
	return c.Name != ""
}
