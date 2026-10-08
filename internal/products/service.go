package products

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/categories"
	"github.com/forgestock/forgestock/internal/platform/logger"
)

type Service struct {
	repo      *Repository
	categoryRepo *categories.Repository
	log       *logger.Logger
}

func NewService(repo *Repository, categoryRepo *categories.Repository, log *logger.Logger) *Service {
	return &Service{repo: repo, categoryRepo: categoryRepo, log: log}
}

type CreateProductRequest struct {
	SKU         string  `json:"sku" validate:"required,min=1,max=50"`
	Name        string  `json:"name" validate:"required,min=1,max=200"`
	Description string  `json:"description" validate:"omitempty,max=2000"`
	CategoryID  string  `json:"category_id" validate:"required"`
	Price       float64 `json:"price" validate:"required,gte=0"`
	Cost        float64 `json:"cost" validate:"required,gte=0"`
	Currency    string  `json:"currency" validate:"required,len=3"`
}

type UpdateProductRequest struct {
	SKU         string   `json:"sku" validate:"omitempty,min=1,max=50"`
	Name        string   `json:"name" validate:"omitempty,min=1,max=200"`
	Description string   `json:"description" validate:"omitempty,max=2000"`
	CategoryID  string   `json:"category_id" validate:"omitempty"`
	Price       *float64 `json:"price" validate:"omitempty,gte=0"`
	Cost        *float64 `json:"cost" validate:"omitempty,gte=0"`
	Currency    string   `json:"currency" validate:"omitempty,len=3"`
	Active      *bool    `json:"active"`
}

func (s *Service) Create(ctx context.Context, req CreateProductRequest) (*Product, error) {
	exists, err := s.repo.ExistsBySKU(ctx, req.SKU, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSKUExists
	}

	category, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
	if err != nil {
		return nil, errors.New("invalid category")
	}
	_ = category

	product := &Product{
		ID:          uuid.New().String(),
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		Price:       req.Price,
		Cost:        req.Cost,
		Currency:    req.Currency,
		Active:      true,
	}

	if err := s.repo.Create(ctx, product); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "product_created", "product", product.ID, map[string]interface{}{
		"sku":         product.SKU,
		"name":        product.Name,
		"category_id": product.CategoryID,
		"price":       product.Price,
	})

	return product, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Product, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateProductRequest) (*Product, error) {
	product, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.SKU != "" && req.SKU != product.SKU {
		exists, err := s.repo.ExistsBySKU(ctx, req.SKU, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrSKUExists
		}
		product.SKU = req.SKU
	}

	if req.Name != "" {
		product.Name = req.Name
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.CategoryID != "" {
		category, err := s.categoryRepo.GetByID(ctx, req.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category")
		}
		_ = category
		product.CategoryID = req.CategoryID
	}
	if req.Price != nil {
		product.Price = *req.Price
	}
	if req.Cost != nil {
		product.Cost = *req.Cost
	}
	if req.Currency != "" {
		product.Currency = req.Currency
	}
	if req.Active != nil {
		product.Active = *req.Active
	}

	if err := s.repo.Update(ctx, product); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "product_updated", "product", id, map[string]interface{}{
		"sku":         product.SKU,
		"name":        product.Name,
		"category_id": product.CategoryID,
		"price":       product.Price,
		"active":      product.Active,
	})

	return product, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.log.AuditLog(ctx, "product_deleted", "product", id, nil)
	return nil
}

func (s *Service) List(ctx context.Context, filter ProductFilter, page, pageSize int) ([]*Product, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "DESC"
	}
	return s.repo.List(ctx, filter, page, pageSize)
}