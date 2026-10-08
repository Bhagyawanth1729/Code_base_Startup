package categories

import (
	"context"
	"errors"

	"github.com/forgestock/forgestock/internal/platform/database"
)

var (
	ErrCategoryNotFound   = errors.New("category not found")
	ErrCategoryInUse      = errors.New("category is in use by products")
	ErrCategoryNameExists = errors.New("category name already exists")
)

type Category struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, category *Category) error {
	query := `
		INSERT INTO categories (id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`
	_, err := r.db.Pool.Exec(ctx, query, category.ID, category.Name, category.Description)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Category, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM categories WHERE id = $1`
	category := &Category{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*Category, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM categories WHERE name = $1`
	category := &Category{}
	err := r.db.Pool.QueryRow(ctx, query, name).Scan(
		&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt)
	if err != nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

func (r *Repository) Update(ctx context.Context, category *Category) error {
	query := `
		UPDATE categories SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
	`
	result, err := r.db.Pool.Exec(ctx, query, category.Name, category.Description, category.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	checkQuery := `SELECT COUNT(*) FROM products WHERE category_id = $1`
	var count int
	if err := r.db.Pool.QueryRow(ctx, checkQuery, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return ErrCategoryInUse
	}

	query := `DELETE FROM categories WHERE id = $1`
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrCategoryNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, page, pageSize int) ([]*Category, int, error) {
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM categories`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, name, description, created_at, updated_at
		FROM categories ORDER BY name ASC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var categories []*Category
	for rows.Next() {
		category := &Category{}
		if err := rows.Scan(&category.ID, &category.Name, &category.Description, &category.CreatedAt, &category.UpdatedAt); err != nil {
			return nil, 0, err
		}
		categories = append(categories, category)
	}

	return categories, total, nil
}