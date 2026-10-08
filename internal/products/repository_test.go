package products

import (
	"context"
	"testing"

	"github.com/forgestock/forgestock/internal/categories"
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

func createTestCategory(t *testing.T, repo *categories.Repository) *categories.Category {
	cat := &categories.Category{
		ID:          "test-category-1",
		Name:        "Test Category",
		Description: "Test Description",
	}
	err := repo.Create(context.Background(), cat)
	require.NoError(t, err)
	return cat
}

func TestProductRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	catRepo := categories.NewRepository(db)
	cat := createTestCategory(t, catRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	product := &Product{
		ID:          "test-product-1",
		SKU:         "TEST-001",
		Name:        "Test Product",
		Description: "Test Description",
		CategoryID:  cat.ID,
		Price:       99.99,
		Cost:        50.00,
		Currency:    "USD",
		Active:      true,
	}

	err := repo.Create(ctx, product)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, product.ID, found.ID)
	assert.Equal(t, product.SKU, found.SKU)
	assert.Equal(t, product.Name, found.Name)
	assert.Equal(t, product.CategoryID, found.CategoryID)
	assert.Equal(t, product.Price, found.Price)
	assert.Equal(t, product.Cost, found.Cost)
	assert.Equal(t, product.Currency, found.Currency)
	assert.Equal(t, product.Active, found.Active)
}

func TestProductRepository_GetBySKU(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	catRepo := categories.NewRepository(db)
	cat := createTestCategory(t, catRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	product := &Product{
		ID:          "test-product-2",
		SKU:         "TEST-002",
		Name:        "Test Product 2",
		Description: "Test Description",
		CategoryID:  cat.ID,
		Price:       149.99,
		Cost:        75.00,
		Currency:    "USD",
		Active:      true,
	}

	err := repo.Create(ctx, product)
	require.NoError(t, err)

	found, err := repo.GetBySKU(ctx, "TEST-002")
	require.NoError(t, err)
	assert.Equal(t, product.ID, found.ID)
	assert.Equal(t, product.SKU, found.SKU)
}

func TestProductRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	catRepo := categories.NewRepository(db)
	cat := createTestCategory(t, catRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	product := &Product{
		ID:          "test-product-3",
		SKU:         "TEST-003",
		Name:        "Original Name",
		Description: "Original Description",
		CategoryID:  cat.ID,
		Price:       199.99,
		Cost:        100.00,
		Currency:    "USD",
		Active:      true,
	}

	err := repo.Create(ctx, product)
	require.NoError(t, err)

	product.Name = "Updated Name"
	product.Price = 249.99
	product.Active = false

	err = repo.Update(ctx, product)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, product.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, 249.99, found.Price)
	assert.False(t, found.Active)
}

func TestProductRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	catRepo := categories.NewRepository(db)
	cat := createTestCategory(t, catRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	product := &Product{
		ID:          "test-product-4",
		SKU:         "TEST-004",
		Name:        "To Delete",
		Description: "Will be deleted",
		CategoryID:  cat.ID,
		Price:       49.99,
		Cost:        25.00,
		Currency:    "USD",
		Active:      true,
	}

	err := repo.Create(ctx, product)
	require.NoError(t, err)

	err = repo.Delete(ctx, product.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, product.ID)
	assert.Error(t, err)
	assert.Equal(t, ErrProductNotFound, err)
}

func TestProductRepository_ListWithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	catRepo := categories.NewRepository(db)
	cat := createTestCategory(t, catRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		product := &Product{
			ID:          "list-product-" + string(rune('0'+i)),
			SKU:         "LIST-" + string(rune('0'+i)),
			Name:        "List Product " + string(rune('0'+i)),
			Description: "Description",
			CategoryID:  cat.ID,
			Price:       10.00 + float64(i)*10,
			Cost:        5.00 + float64(i)*5,
			Currency:    "USD",
			Active:      true,
		}
		err := repo.Create(ctx, product)
		require.NoError(t, err)
	}

	filter := ProductFilter{
		CategoryID: cat.ID,
		Active:     boolPtr(true),
	}
	products, total, err := repo.List(ctx, filter, 1, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 3)
	assert.LessOrEqual(t, len(products), 10)
}

func TestProductRepository_ExistsBySKU(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	catRepo := categories.NewRepository(db)
	cat := createTestCategory(t, catRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	product := &Product{
		ID:          "sku-test-1",
		SKU:         "SKU-TEST",
		Name:        "SKU Test",
		Description: "Test",
		CategoryID:  cat.ID,
		Price:       10.00,
		Cost:        5.00,
		Currency:    "USD",
		Active:      true,
	}

	err := repo.Create(ctx, product)
	require.NoError(t, err)

	exists, err := repo.ExistsBySKU(ctx, "SKU-TEST", "")
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = repo.ExistsBySKU(ctx, "SKU-TEST", "sku-test-1")
	require.NoError(t, err)
	assert.False(t, exists)

	exists, err = repo.ExistsBySKU(ctx, "NONEXISTENT", "")
	require.NoError(t, err)
	assert.False(t, exists)
}

func boolPtr(b bool) *bool {
	return &b
}