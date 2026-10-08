package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTManager_GenerateAndValidateAccessToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)

	token, err := manager.GenerateAccessToken("user-123", "test@example.com", []string{"admin"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, []string{"admin"}, claims.Roles)
	assert.Equal(t, "access", claims.TokenType)
	assert.NotEmpty(t, claims.ID)
}

func TestJWTManager_GenerateAndValidateRefreshToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)

	token, err := manager.GenerateRefreshToken("user-123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := manager.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "refresh", claims.TokenType)
	assert.NotEmpty(t, claims.ID)
}

func TestJWTManager_ValidateExpiredToken(t *testing.T) {
	manager := NewJWTManager("test-secret", -1*time.Hour, 24*time.Hour)

	token, err := manager.GenerateAccessToken("user-123", "test@example.com", []string{"admin"})
	require.NoError(t, err)

	_, err = manager.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Equal(t, ErrExpiredToken, err)
}

func TestJWTManager_InvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)

	_, err := manager.ValidateAccessToken("invalid-token")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestJWTManager_WrongTokenType(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)

	refreshToken, err := manager.GenerateRefreshToken("user-123")
	require.NoError(t, err)

	_, err = manager.ValidateAccessToken(refreshToken)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidToken, err)
}

func TestHashPassword(t *testing.T) {
	password := "secure-password-123"
	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestCheckPasswordHash(t *testing.T) {
	password := "secure-password-123"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	assert.True(t, CheckPasswordHash(password, hash))
	assert.False(t, CheckPasswordHash("wrong-password", hash))
}

func TestClaims_UniqueID(t *testing.T) {
	manager := NewJWTManager("test-secret", 15*time.Minute, 24*time.Hour)

	token1, err := manager.GenerateAccessToken("user-123", "test@example.com", []string{"admin"})
	require.NoError(t, err)

	token2, err := manager.GenerateAccessToken("user-123", "test@example.com", []string{"admin"})
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)

	claims1, _ := manager.ValidateAccessToken(token1)
	claims2, _ := manager.ValidateAccessToken(token2)

	assert.NotEqual(t, claims1.ID, claims2.ID)
	assert.True(t, uuid.Validate(claims1.ID) == nil)
	assert.True(t, uuid.Validate(claims2.ID) == nil)
}