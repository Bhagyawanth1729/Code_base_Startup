package users

import (
	"context"
	"errors"

	"github.com/forgestock/forgestock/internal/platform/logger"
)

type Service struct {
	repo *Repository
	log  *logger.Logger
}

func NewService(repo *Repository, log *logger.Logger) *Service {
	return &Service{repo: repo, log: log}
}

type UpdateProfileRequest struct {
	FirstName string `json:"first_name" validate:"omitempty,min=1,max=100"`
	LastName  string `json:"last_name" validate:"omitempty,min=1,max=100"`
}

func (s *Service) GetProfile(ctx context.Context, userID string) (*User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (*User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "profile_updated", "user", userID, map[string]interface{}{
		"first_name": user.FirstName,
		"last_name":  user.LastName,
	})

	return user, nil
}

func (s *Service) DeactivateUser(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	user.Active = false
	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	s.log.AuditLog(ctx, "user_deactivated", "user", userID, nil)
	return nil
}

func (s *Service) ActivateUser(ctx context.Context, userID string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	user.Active = true
	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	s.log.AuditLog(ctx, "user_activated", "user", userID, nil)
	return nil
}

func (s *Service) ListUsers(ctx context.Context, page, pageSize int) ([]*User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, page, pageSize)
}

func (s *Service) GetUserByID(ctx context.Context, userID string) (*User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *Service) UpdateUserRole(ctx context.Context, userID, role string) error {
	validRoles := map[string]bool{
		"admin": true, "manager": true, "warehouse_operator": true, "customer": true,
	}
	if !validRoles[role] {
		return errors.New("invalid role")
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	user.Role = role
	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	s.log.AuditLog(ctx, "user_role_changed", "user", userID, map[string]interface{}{
		"new_role": role,
	})
	return nil
}