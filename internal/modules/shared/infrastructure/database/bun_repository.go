package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"

	_ "github.com/go-sql-driver/mysql"

	"vision-api-app/internal/config"
	"vision-api-app/internal/modules/household/domain/entity"
)

// Receipt BUNモデル
type Receipt struct {
	bun.BaseModel `bun:"table:receipts"`

	ID            string    `bun:"id,pk,type:varchar(36)"`
	StoreName     string    `bun:"store_name,notnull"`
	PurchaseDate  time.Time `bun:"purchase_date,notnull"`
	TotalAmount   int       `bun:"total_amount,notnull"`
	TaxAmount     int       `bun:"tax_amount,notnull,default:0"`
	PaymentMethod string    `bun:"payment_method,type:varchar(50),default:''"`
	ReceiptNumber string    `bun:"receipt_number,type:varchar(100),default:''"`
	Category      *string   `bun:"category,type:varchar(50)"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     time.Time `bun:"updated_at,notnull,default:current_timestamp"`

	Items []ReceiptItem `bun:"rel:has-many,join:id=receipt_id"`
}

// ReceiptItem BUNモデル
type ReceiptItem struct {
	bun.BaseModel `bun:"table:receipt_items"`

	ID        string    `bun:"id,pk,type:varchar(36)"`
	ReceiptID string    `bun:"receipt_id,notnull"`
	Name      string    `bun:"name,notnull"`
	Quantity  int       `bun:"quantity,notnull,default:1"`
	Price     int       `bun:"price,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

// ExpenseEntry BUNモデル
type ExpenseEntry struct {
	bun.BaseModel `bun:"table:expense_entries"`

	ID          string    `bun:"id,pk,type:varchar(36)"`
	ReceiptID   *string   `bun:"receipt_id,type:varchar(36)"`
	Date        time.Time `bun:"date,notnull"`
	Category    string    `bun:"category,notnull,type:varchar(50)"`
	Amount      int       `bun:"amount,notnull"`
	Description *string   `bun:"description,type:text"`
	Tags        []string  `bun:"tags,type:json"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}

// Category BUNモデル
type Category struct {
	bun.BaseModel `bun:"table:categories"`

	ID          string    `bun:"id,pk,type:varchar(36)"`
	Name        string    `bun:"name,notnull,unique,type:varchar(50)"`
	Description *string   `bun:"description,type:text"`
	Color       *string   `bun:"color,type:varchar(7)"`
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

// BunReceiptRepository BUN実装
type BunReceiptRepository struct {
	db *bun.DB
}

// NewBunReceiptRepository 新しいBunReceiptRepositoryを作成
func NewBunReceiptRepository(cfg *config.MySQLConfig) (*BunReceiptRepository, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := bun.NewDB(sqldb, mysqldialect.New())

	// 接続確認
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &BunReceiptRepository{db: db}, nil
}

// NewBunReceiptRepositoryWithDB DBインスタンスから作成（テスト用）
func NewBunReceiptRepositoryWithDB(db *bun.DB) *BunReceiptRepository {
	return &BunReceiptRepository{db: db}
}

// Create レシートを作成
func (r *BunReceiptRepository) Create(ctx context.Context, receipt *entity.Receipt) error {
	model := r.toModel(receipt)

	// トランザクション内で実行
	return r.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(model).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create receipt: %w", err)
		}

		if len(model.Items) > 0 {
			if _, err := tx.NewInsert().Model(&model.Items).Exec(ctx); err != nil {
				return fmt.Errorf("failed to create receipt items: %w", err)
			}
		}

		return nil
	})
}

// FindByID IDでレシートを検索
func (r *BunReceiptRepository) FindByID(ctx context.Context, id string) (*entity.Receipt, error) {
	model := &Receipt{}
	err := r.db.NewSelect().
		Model(model).
		Relation("Items").
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("receipt not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find receipt: %w", err)
	}

	return r.toEntity(model), nil
}

// FindAll 全レシートを取得
func (r *BunReceiptRepository) FindAll(ctx context.Context, limit, offset int) ([]*entity.Receipt, error) {
	var models []Receipt
	query := r.db.NewSelect().
		Model(&models).
		Relation("Items").
		Order("purchase_date DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to find receipts: %w", err)
	}

	receipts := make([]*entity.Receipt, len(models))
	for i, model := range models {
		receipts[i] = r.toEntity(&model)
	}
	return receipts, nil
}

// FindByDateRange 日付範囲でレシートを検索
func (r *BunReceiptRepository) FindByDateRange(ctx context.Context, start, end time.Time) ([]*entity.Receipt, error) {
	var models []Receipt
	err := r.db.NewSelect().
		Model(&models).
		Relation("Items").
		Where("purchase_date BETWEEN ? AND ?", start, end).
		Order("purchase_date DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to find receipts by date range: %w", err)
	}

	receipts := make([]*entity.Receipt, len(models))
	for i, model := range models {
		receipts[i] = r.toEntity(&model)
	}
	return receipts, nil
}

// Update レシートを更新
func (r *BunReceiptRepository) Update(ctx context.Context, receipt *entity.Receipt) error {
	model := r.toModel(receipt)
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update receipt: %w", err)
	}
	return nil
}

// Delete レシートを削除
func (r *BunReceiptRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*Receipt)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete receipt: %w", err)
	}
	return nil
}

// Close データベース接続を閉じる
func (r *BunReceiptRepository) Close() error {
	return r.db.Close()
}

// toModel エンティティをモデルに変換
func (r *BunReceiptRepository) toModel(receipt *entity.Receipt) *Receipt {
	model := &Receipt{
		ID:            receipt.ID,
		StoreName:     receipt.StoreName,
		PurchaseDate:  receipt.PurchaseDate,
		TotalAmount:   receipt.TotalAmount,
		TaxAmount:     receipt.TaxAmount,
		PaymentMethod: receipt.PaymentMethod,
		ReceiptNumber: receipt.ReceiptNumber,
		CreatedAt:     receipt.CreatedAt,
		UpdatedAt:     receipt.UpdatedAt,
	}

	if receipt.Category != "" {
		model.Category = &receipt.Category
	}

	for _, item := range receipt.Items {
		model.Items = append(model.Items, ReceiptItem{
			ID:        item.ID,
			ReceiptID: item.ReceiptID,
			Name:      item.Name,
			Quantity:  item.Quantity,
			Price:     item.Price,
			CreatedAt: item.CreatedAt,
		})
	}

	return model
}

// toEntity モデルをエンティティに変換
func (r *BunReceiptRepository) toEntity(model *Receipt) *entity.Receipt {
	receipt := &entity.Receipt{
		ID:            model.ID,
		StoreName:     model.StoreName,
		PurchaseDate:  model.PurchaseDate,
		TotalAmount:   model.TotalAmount,
		TaxAmount:     model.TaxAmount,
		PaymentMethod: model.PaymentMethod,
		ReceiptNumber: model.ReceiptNumber,
		CreatedAt:     model.CreatedAt,
		UpdatedAt:     model.UpdatedAt,
		Items:         []entity.ReceiptItem{},
	}

	if model.Category != nil {
		receipt.Category = *model.Category
	}

	for _, itemModel := range model.Items {
		receipt.Items = append(receipt.Items, entity.ReceiptItem{
			ID:        itemModel.ID,
			ReceiptID: itemModel.ReceiptID,
			Name:      itemModel.Name,
			Quantity:  itemModel.Quantity,
			Price:     itemModel.Price,
			CreatedAt: itemModel.CreatedAt,
		})
	}

	return receipt
}

// BunExpenseRepository BUN実装
type BunExpenseRepository struct {
	db *bun.DB
}

// NewBunExpenseRepository 新しいBunExpenseRepositoryを作成
func NewBunExpenseRepository(cfg *config.MySQLConfig) (*BunExpenseRepository, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := bun.NewDB(sqldb, mysqldialect.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &BunExpenseRepository{db: db}, nil
}

// NewBunExpenseRepositoryWithDB DBインスタンスから作成（テスト用）
func NewBunExpenseRepositoryWithDB(db *bun.DB) *BunExpenseRepository {
	return &BunExpenseRepository{db: db}
}

// Create 家計簿エントリを作成
func (r *BunExpenseRepository) Create(ctx context.Context, entry *entity.ExpenseEntry) error {
	model, err := r.toExpenseModel(entry)
	if err != nil {
		return fmt.Errorf("failed to convert to model: %w", err)
	}

	_, err = r.db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create expense entry: %w", err)
	}
	return nil
}

// FindByID IDで家計簿エントリを検索
func (r *BunExpenseRepository) FindByID(ctx context.Context, id string) (*entity.ExpenseEntry, error) {
	model := &ExpenseEntry{}
	err := r.db.NewSelect().
		Model(model).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("expense entry not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find expense entry: %w", err)
	}

	return r.toExpenseEntity(model)
}

// FindAll 全家計簿エントリを取得
func (r *BunExpenseRepository) FindAll(ctx context.Context, limit, offset int) ([]*entity.ExpenseEntry, error) {
	var models []ExpenseEntry
	query := r.db.NewSelect().
		Model(&models).
		Order("date DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to find expense entries: %w", err)
	}

	entries := make([]*entity.ExpenseEntry, len(models))
	for i, model := range models {
		entry, err := r.toExpenseEntity(&model)
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

// FindByDateRange 日付範囲で家計簿エントリを検索
func (r *BunExpenseRepository) FindByDateRange(ctx context.Context, start, end time.Time) ([]*entity.ExpenseEntry, error) {
	var models []ExpenseEntry
	err := r.db.NewSelect().
		Model(&models).
		Where("date BETWEEN ? AND ?", start, end).
		Order("date DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to find expense entries by date range: %w", err)
	}

	entries := make([]*entity.ExpenseEntry, len(models))
	for i, model := range models {
		entry, err := r.toExpenseEntity(&model)
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

// FindByCategory カテゴリで家計簿エントリを検索
func (r *BunExpenseRepository) FindByCategory(ctx context.Context, category string) ([]*entity.ExpenseEntry, error) {
	var models []ExpenseEntry
	err := r.db.NewSelect().
		Model(&models).
		Where("category = ?", category).
		Order("date DESC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to find expense entries by category: %w", err)
	}

	entries := make([]*entity.ExpenseEntry, len(models))
	for i, model := range models {
		entry, err := r.toExpenseEntity(&model)
		if err != nil {
			return nil, err
		}
		entries[i] = entry
	}
	return entries, nil
}

// Update 家計簿エントリを更新
func (r *BunExpenseRepository) Update(ctx context.Context, entry *entity.ExpenseEntry) error {
	model, err := r.toExpenseModel(entry)
	if err != nil {
		return fmt.Errorf("failed to convert to model: %w", err)
	}

	_, err = r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update expense entry: %w", err)
	}
	return nil
}

// Delete 家計簿エントリを削除
func (r *BunExpenseRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*ExpenseEntry)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete expense entry: %w", err)
	}
	return nil
}

// Close データベース接続を閉じる
func (r *BunExpenseRepository) Close() error {
	return r.db.Close()
}

// toExpenseModel エンティティをモデルに変換
func (r *BunExpenseRepository) toExpenseModel(entry *entity.ExpenseEntry) (*ExpenseEntry, error) {
	model := &ExpenseEntry{
		ID:        entry.ID,
		Date:      entry.Date,
		Category:  entry.Category,
		Amount:    entry.Amount,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
		Tags:      entry.Tags,
	}

	if entry.ReceiptID != nil {
		model.ReceiptID = entry.ReceiptID
	}

	if entry.Description != "" {
		model.Description = &entry.Description
	}

	// Tagsが nil の場合は空配列に
	if model.Tags == nil {
		model.Tags = []string{}
	}

	return model, nil
}

// toExpenseEntity モデルをエンティティに変換
func (r *BunExpenseRepository) toExpenseEntity(model *ExpenseEntry) (*entity.ExpenseEntry, error) {
	entry := &entity.ExpenseEntry{
		ID:        model.ID,
		Date:      model.Date,
		Category:  model.Category,
		Amount:    model.Amount,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
		Tags:      model.Tags,
	}

	if model.ReceiptID != nil {
		entry.ReceiptID = model.ReceiptID
	}

	if model.Description != nil {
		entry.Description = *model.Description
	}

	// Tagsが nil の場合は空配列に
	if entry.Tags == nil {
		entry.Tags = []string{}
	}

	return entry, nil
}

// BunCategoryRepository BUN実装
type BunCategoryRepository struct {
	db *bun.DB
}

// NewBunCategoryRepository 新しいBunCategoryRepositoryを作成
func NewBunCategoryRepository(cfg *config.MySQLConfig) (*BunCategoryRepository, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := bun.NewDB(sqldb, mysqldialect.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &BunCategoryRepository{db: db}, nil
}

// NewBunCategoryRepositoryWithDB DBインスタンスから作成（テスト用）
func NewBunCategoryRepositoryWithDB(db *bun.DB) *BunCategoryRepository {
	return &BunCategoryRepository{db: db}
}

// Create カテゴリを作成
func (r *BunCategoryRepository) Create(ctx context.Context, category *entity.Category) error {
	model := r.toCategoryModel(category)
	_, err := r.db.NewInsert().Model(model).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create category: %w", err)
	}
	return nil
}

// FindByID IDでカテゴリを検索
func (r *BunCategoryRepository) FindByID(ctx context.Context, id string) (*entity.Category, error) {
	model := &Category{}
	err := r.db.NewSelect().
		Model(model).
		Where("id = ?", id).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("category not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	return r.toCategoryEntity(model), nil
}

// FindAll 全カテゴリを取得
func (r *BunCategoryRepository) FindAll(ctx context.Context) ([]*entity.Category, error) {
	var models []Category
	err := r.db.NewSelect().
		Model(&models).
		Order("name ASC").
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to find categories: %w", err)
	}

	categories := make([]*entity.Category, len(models))
	for i, model := range models {
		categories[i] = r.toCategoryEntity(&model)
	}
	return categories, nil
}

// FindByName 名前でカテゴリを検索
func (r *BunCategoryRepository) FindByName(ctx context.Context, name string) (*entity.Category, error) {
	model := &Category{}
	err := r.db.NewSelect().
		Model(model).
		Where("name = ?", name).
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("category not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find category: %w", err)
	}

	return r.toCategoryEntity(model), nil
}

// Update カテゴリを更新
func (r *BunCategoryRepository) Update(ctx context.Context, category *entity.Category) error {
	model := r.toCategoryModel(category)
	_, err := r.db.NewUpdate().
		Model(model).
		WherePK().
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to update category: %w", err)
	}
	return nil
}

// Delete カテゴリを削除
func (r *BunCategoryRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.NewDelete().
		Model((*Category)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}
	return nil
}

// Close データベース接続を閉じる
func (r *BunCategoryRepository) Close() error {
	return r.db.Close()
}

// toCategoryModel エンティティをモデルに変換
func (r *BunCategoryRepository) toCategoryModel(category *entity.Category) *Category {
	model := &Category{
		ID:        category.ID,
		Name:      category.Name,
		CreatedAt: category.CreatedAt,
	}

	if category.Description != "" {
		model.Description = &category.Description
	}

	if category.Color != "" {
		model.Color = &category.Color
	}

	return model
}

// toCategoryEntity モデルをエンティティに変換
func (r *BunCategoryRepository) toCategoryEntity(model *Category) *entity.Category {
	category := &entity.Category{
		ID:        model.ID,
		Name:      model.Name,
		CreatedAt: model.CreatedAt,
	}

	if model.Description != nil {
		category.Description = *model.Description
	}

	if model.Color != nil {
		category.Color = *model.Color
	}

	return category
}
