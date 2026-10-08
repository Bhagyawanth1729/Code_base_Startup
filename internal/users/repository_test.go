package users

import (
	"context"
	"testing"

	"github.com/forgestock/forgestock/internal/platform/config"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *database.DB {
	cfg := &config.Config{
		DatabaseURL: "postgres://forgestock:forgestock@localhost:5432/forgestock_test?sslmode=disable",
	}
	log := logger.New("debug", true)
	db, err := database.New(cfg, log)
	require.NoError(t, err)
	return db
}

func TestUserRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user := &User{
		ID:        "test-user-1",
		Email:     "test@example.com",
		Password:  "hashed-password",
		FirstName: "John",
		LastName:  "Doe",
		Role:      "customer",
		Active:    true,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.Email, found.Email)
	assert.Equal(t, user.FirstName, found.FirstName)
	assert.Equal(t, user.LastName, found.LastName)
	assert.Equal(t, user.Role, found.Role)
	assert.Equal(t, user.Active, found.Active)
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user := &User{
		ID:        "test-user-2",
		Email:     "unique@example.com",
		Password:  "hashed-password",
		FirstName: "Jane",
		LastName:  "Smith",
		Role:      "customer",
		Active:    true,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	found, err := repo.GetByEmail(ctx, "unique@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, user.Email, found.Email)
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nonexistent@example.com")
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user := &User{
		ID:        "test-user-3",
		Email:     "update@example.com",
		Password:  "hashed-password",
		FirstName: "Original",
		LastName:  "Name",
		Role:      "customer",
		Active:    true,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	user.FirstName = "Updated"
	user.LastName = "Name"
	user.Role = "manager"

	err = repo.Update(ctx, user)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", found.FirstName)
	assert.Equal(t, "manager", found.Role)
}

func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user := &User{
		ID:        "test-user-4",
		Email:     "delete@example.com",
		Password:  "hashed-password",
		FirstName: "To",
		LastName:  "Delete",
		Role:      "customer",
		Active:    true,
	}

	err := repo.Create(ctx, user)
	require.NoError(t, err)

	err = repo.Delete(ctx, user.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, user.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
}

func TestUserRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		user := &User{
			ID:        "list-user-" + string(rune('0'+i)),
			Email:     "list" + string(rune('0'+i)) + "@example.com",
			Password:  "hashed-password",
			FirstName: "User",
			LastName:  string(rune('0'+i)),
			Role:      "customer",
			Active:    true,
		}
		err := repo.Create(ctx, user)
		require.NoError(t, err)
	}

	users, total, err := repo.List(ctx, 1, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 5)
	assert.LessOrEqual(t, len(users), 10)
}

func TestUserRepository_EmailUniqueConstraint(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	user1 := &User{
		ID:        "unique-1",
		Email:     "duplicate@example.com",
		Password:  "hashed-password",
		FirstName: "User",
		LastName:  "One",
		Role:      "customer",
		Active:    true,
	}

	err := repo.Create(ctx, user1)
	require.NoError(t, err)

	user2 := &User{
		ID:        "unique-2",
		Email:     "duplicate@example.com",
		Password:  "hashed-password",
		FirstName: "User",
		LastName:  "Two",
		Role:      "customer",
		Active:    true,
	}

	err = repo.Create(ctx, user2)
	assert.Error(t, err)
}