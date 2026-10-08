package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/forgestock/forgestock/internal/platform/redis"
	"github.com/forgestock/forgestock/internal/users"
)

type Service struct {
	db           *database.DB
	redis        *redis.Client
	userRepo     *users.Repository
	jwtManager   *JWTManager
	log          *logger.Logger
	refreshTokenTTL time.Duration
}

func NewService(
	db *database.DB,
	redisClient *redis.Client,
	userRepo *users.Repository,
	jwtManager *JWTManager,
	log *logger.Logger,
	refreshTokenTTL time.Duration,
) *Service {
	return &Service{
		db:           db,
		redis:        redisClient,
		userRepo:     userRepo,
		jwtManager:   jwtManager,
		log:          log,
		refreshTokenTTL: refreshTokenTTL,
	}
}

type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8,max=72"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
	Role      string `json:"role" validate:"omitempty,oneof=admin manager warehouse_operator customer"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*TokenResponse, error) {
	role := req.Role
	if role == "" {
		role = "customer"
	}

	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, users.ErrUserNotFound) {
		return nil, err
	}
	if existingUser != nil {
		return nil, users.ErrEmailExists
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &users.User{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Password:  hashedPassword,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      role,
		Active:    true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "user_registered", "user", user.ID, map[string]interface{}{
		"email": req.Email,
		"role":  role,
	})

	return s.generateTokens(ctx, user)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*TokenResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.Active {
		return nil, ErrInvalidCredentials
	}

	if !CheckPasswordHash(req.Password, user.Password) {
		s.log.AuditLog(ctx, "login_failed", "user", user.ID, map[string]interface{}{
			"reason": "invalid_password",
		})
		return nil, ErrInvalidCredentials
	}

	s.log.AuditLog(ctx, "login", "user", user.ID, map[string]interface{}{
		"email": user.Email,
	})

	return s.generateTokens(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (*TokenResponse, error) {
	claims, err := s.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrExpiredToken) {
			return nil, errors.New("refresh token expired")
		}
		return nil, errors.New("invalid refresh token")
	}

	revoked, err := s.redis.Exists(ctx, "revoked_token:"+claims.ID).Result()
	if err != nil {
		return nil, err
	}
	if revoked > 0 {
		return nil, ErrTokenRevoked
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	if !user.Active {
		return nil, errors.New("user account deactivated")
	}

	if err := s.redis.Set(ctx, "revoked_token:"+claims.ID, "1", s.refreshTokenTTL).Err(); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "token_refreshed", "user", user.ID, nil)

	return s.generateTokens(ctx, user)
}

func (s *Service) Logout(ctx context.Context, accessToken string) error {
	claims, err := s.jwtManager.ValidateAccessToken(accessToken)
	if err != nil {
		return err
	}

	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		if err := s.redis.Set(ctx, "revoked_token:"+claims.ID, "1", ttl).Err(); err != nil {
			return err
		}
	}

	s.log.AuditLog(ctx, "logout", "user", claims.UserID, nil)
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !CheckPasswordHash(currentPassword, user.Password) {
		return ErrInvalidCredentials
	}

	hashedPassword, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	user.Password = hashedPassword
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	s.log.AuditLog(ctx, "password_changed", "user", userID, nil)

	if err := s.revokeAllUserTokens(ctx, userID); err != nil {
		return err
	}

	return nil
}

func (s *Service) generateTokens(ctx context.Context, user *users.User) (*TokenResponse, error) {
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, []string{user.Role})
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, err
	}

	refreshClaims, _ := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err := s.redis.Set(ctx, "refresh_token:"+refreshClaims.ID, user.ID, s.refreshTokenTTL).Err(); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.jwtManager.accessTokenTTL.Seconds()),
	}, nil
}

func (s *Service) revokeAllUserTokens(ctx context.Context, userID string) error {
	pattern := "refresh_token:*"
	iter := s.redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		storedUserID, err := s.redis.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		if storedUserID == userID {
			tokenID := strings.TrimPrefix(key, "refresh_token:")
			s.redis.Set(ctx, "revoked_token:"+tokenID, "1", s.refreshTokenTTL)
		}
	}
	return iter.Err()
}