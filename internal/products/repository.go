package products

import (
	"context"
	"errors"

	"github.com/forgestock/forgestock/internal/platform/database"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrSKUExists       = errors.New("SKU already exists")
)

type Product struct {
	ID          string  `json:"id"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	CategoryID  string  `json:"category_id"`
	Price       float64 `json:"price"`
	Cost        float64 `json:"cost"`
	Currency    string  `json:"currency"`
	Active      bool    `json:"active"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, product *Product) error {
	query := `
		INSERT INTO products (id, sku, name, description, category_id, price, cost, currency, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err := r.db.Pool.Exec(ctx, query,
		product.ID, product.SKU, product.Name, product.Description, product.CategoryID,
		product.Price, product.Cost, product.Currency, product.Active)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Product, error) {
	query := `
		SELECT id, sku, name, description, category_id, price, cost, currency, active, created_at, updated_at
		FROM products WHERE id = $1
	`
	product := &Product{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&product.ID, &product.SKU, &product.Name, &product.Description, &product.CategoryID,
		&product.Price, &product.Cost, &product.Currency, &product.Active, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

func (r *Repository) GetBySKU(ctx context.Context, sku string) (*Product, error) {
	query := `
		SELECT id, sku, name, description, category_id, price, cost, currency, active, created_at, updated_at
		FROM products WHERE sku = $1
	`
	product := &Product{}
	err := r.db.Pool.QueryRow(ctx, query, sku).Scan(
		&product.ID, &product.SKU, &product.Name, &product.Description, &product.CategoryID,
		&product.Price, &product.Cost, &product.Currency, &product.Active, &product.CreatedAt, &product.UpdatedAt)
	if err != nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

func (r *Repository) Update(ctx context.Context, product *Product) error {
	query := `
		UPDATE products SET sku = $1, name = $2, description = $3, category_id = $4,
		price = $5, cost = $6, currency = $7, active = $8, updated_at = NOW()
		WHERE id = $9
	`
	result, err := r.db.Pool.Exec(ctx, query,
		product.SKU, product.Name, product.Description, product.CategoryID,
		product.Price, product.Cost, product.Currency, product.Active, product.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM products WHERE id = $1`
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProductNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, filter ProductFilter, page, pageSize int) ([]*Product, int, error) {
	offset := (page - 1) * pageSize

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.CategoryID != "" {
		whereClause += " AND category_id = $" + string(rune(argIndex+'0'))
		args = append(args, filter.CategoryID)
		argIndex++
	}
	if filter.Active != nil {
		whereClause += " AND active = $" + string(rune(argIndex+'0'))
		args = append(args, *filter.Active)
		argIndex++
	}
	if filter.Search != "" {
		whereClause += " AND (name ILIKE $" + string(rune(argIndex+'0')) + " OR sku ILIKE $" + string(rune(argIndex+'0')) + " OR description ILIKE $" + string(rune(argIndex+'0')) + ")"
		args = append(args, "%"+filter.Search+"%")
		argIndex++
	}

	countQuery := "SELECT COUNT(*) FROM products " + whereClause
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "created_at DESC"
	if filter.SortBy != "" {
		orderBy = filter.SortBy + " " + filter.SortOrder
	}

	query := "SELECT id, sku, name, description, category_id, price, cost, currency, active, created_at, updated_at FROM products " + whereClause + " ORDER BY " + orderBy + " LIMIT $" + string(rune(argIndex+'0')) + " OFFSET $" + string(rune(argIndex+1+'0'))
	args = append(args, pageSize, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var products []*Product
	for rows.Next() {
		product := &Product{}
		if err := rows.Scan(
			&product.ID, &product.SKU, &product.Name, &product.Description, &product.CategoryID,
			&product.Price, &product.Cost, &product.Currency, &product.Active, &product.CreatedAt, &product.UpdatedAt); err != nil {
			return nil, 0, err
		}
		products = append(products, product)
	}

	return products, total, nil
}

func (r *Repository) ExistsBySKU(ctx context.Context, sku string, excludeID string) (bool, error) {
	query := `SELECT COUNT(*) FROM products WHERE sku = $1`
	args := []interface{}{sku}
	if excludeID != "" {
		query += " AND id != $2"
		args = append(args, excludeID)
	}
	var count int
	err := r.db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	return count > 0, err
}

type ProductFilter struct {
	CategoryID string
	Active     *bool
	Search     string
	SortBy     string
	SortOrder  string
}