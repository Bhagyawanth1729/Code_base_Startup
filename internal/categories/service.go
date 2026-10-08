package categories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/platform/logger"
)

type Service struct {
	repo *Repository
	log  *logger.Logger
}

func NewService(repo *Repository, log *logger.Logger) *Service {
	return &Service{repo: repo, log: log}
}

type CreateCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name" validate:"omitempty,min=1,max=100"`
	Description string `json:"description" validate:"omitempty,max=500"`
}

func (s *Service) Create(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	existing, err := s.repo.GetByName(ctx, req.Name)
	if err == nil && existing != nil {
		return nil, ErrCategoryNameExists
	}

	category := &Category{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.Create(ctx, category); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "category_created", "category", category.ID, map[string]interface{}{
		"name": category.Name,
	})

	return category, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Category, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateCategoryRequest) (*Category, error) {
	category, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" && req.Name != category.Name {
		existing, err := s.repo.GetByName(ctx, req.Name)
		if err == nil && existing != nil && existing.ID != id {
			return nil, ErrCategoryNameExists
		}
		category.Name = req.Name
	}

	if req.Description != "" {
		category.Description = req.Description
	}

	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "category_updated", "category", id, map[string]interface{}{
		"name":        category.Name,
		"description": category.Description,
	})

	return category, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.log.AuditLog(ctx, "category_deleted", "category", id, nil)
	return nil
}

func (s *Service) List(ctx context.Context, page, pageSize int) ([]*Category, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, page, pageSize)
}