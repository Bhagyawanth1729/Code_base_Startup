package inventory

import (
	"context"
	"testing"

	"github.com/forgestock/forgestock/internal/platform/config"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/forgestock/forgestock/internal/products"
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

func createTestProduct(t *testing.T, repo *products.Repository) *products.Product {
	product := &products.Product{
		ID:          "test-product-inv-1",
		SKU:         "INV-001",
		Name:        "Inventory Test Product",
		Description: "Test",
		CategoryID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		Price:       99.99,
		Cost:        50.00,
		Currency:    "USD",
		Active:      true,
	}
	err := repo.Create(context.Background(), product)
	require.NoError(t, err)
	return product
}

func TestInventoryRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-1",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     100,
		ReservedQty:      0,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, inv.ID, found.ID)
	assert.Equal(t, inv.ProductID, found.ProductID)
	assert.Equal(t, inv.WarehouseID, found.WarehouseID)
	assert.Equal(t, inv.AvailableQty, found.AvailableQty)
	assert.Equal(t, inv.ReservedQty, found.ReservedQty)
	assert.Equal(t, inv.DamagedQty, found.DamagedQty)
	assert.Equal(t, inv.ReorderThreshold, found.ReorderThreshold)
	assert.Equal(t, inv.Version, found.Version)
}

func TestInventoryRepository_GetByProductAndWarehouse(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-2",
		ProductID:        product.ID,
		WarehouseID:      "warehouse-a",
		AvailableQty:     50,
		ReservedQty:      10,
		DamagedQty:       2,
		ReorderThreshold: 5,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	found, err := repo.GetByProductAndWarehouse(ctx, product.ID, "warehouse-a")
	require.NoError(t, err)
	assert.Equal(t, inv.ID, found.ID)
	assert.Equal(t, inv.AvailableQty, found.AvailableQty)
	assert.Equal(t, inv.ReservedQty, found.ReservedQty)
}

func TestInventoryRepository_ReserveStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-3",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     100,
		ReservedQty:      0,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	err = repo.ReserveStock(ctx, inv.ID, 20, 1)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, 80, found.AvailableQty)
	assert.Equal(t, 20, found.ReservedQty)
	assert.Equal(t, 2, found.Version)
}

func TestInventoryRepository_ReserveStock_Insufficient(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-4",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     10,
		ReservedQty:      0,
		DamagedQty:       0,
		ReorderThreshold: 5,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	err = repo.ReserveStock(ctx, inv.ID, 20, 1)
	assert.Error(t, err)
	assert.Equal(t, ErrOptimisticLock, err)
}

func TestInventoryRepository_ReserveStock_OptimisticLock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-5",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     100,
		ReservedQty:      0,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	// Simulate concurrent update by using wrong version
	err = repo.ReserveStock(ctx, inv.ID, 10, 999)
	assert.Error(t, err)
	assert.Equal(t, ErrOptimisticLock, err)
}

func TestInventoryRepository_ReleaseStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-6",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     80,
		ReservedQty:      20,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	err = repo.ReleaseStock(ctx, inv.ID, 10, 1)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, 90, found.AvailableQty)
	assert.Equal(t, 10, found.ReservedQty)
	assert.Equal(t, 2, found.Version)
}

func TestInventoryRepository_DeductStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-7",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     80,
		ReservedQty:      20,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	err = repo.DeductStock(ctx, inv.ID, 10, 1)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, 80, found.AvailableQty)
	assert.Equal(t, 10, found.ReservedQty)
	assert.Equal(t, 2, found.Version)
}

func TestInventoryRepository_AddStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-8",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     50,
		ReservedQty:      0,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	err = repo.AddStock(ctx, inv.ID, 30, 1)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, 80, found.AvailableQty)
	assert.Equal(t, 2, found.Version)
}

func TestInventoryRepository_AdjustStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-9",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     100,
		ReservedQty:      10,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	err = repo.AdjustStock(ctx, inv.ID, -20, 5, 2, 1)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, 80, found.AvailableQty)
	assert.Equal(t, 15, found.ReservedQty)
	assert.Equal(t, 2, found.DamagedQty)
	assert.Equal(t, 2, found.Version)
}

func TestInventoryRepository_AdjustStock_NegativeInventory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-10",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     10,
		ReservedQty:      0,
		DamagedQty:       0,
		ReorderThreshold: 5,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	err = repo.AdjustStock(ctx, inv.ID, -20, 0, 0, 1)
	assert.Error(t, err)
	assert.Equal(t, ErrOptimisticLock, err)
}

func TestInventoryRepository_CreateTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	inv := &Inventory{
		ID:               "test-inv-11",
		ProductID:        product.ID,
		WarehouseID:      "default",
		AvailableQty:     100,
		ReservedQty:      0,
		DamagedQty:       0,
		ReorderThreshold: 10,
		Version:          1,
	}

	err := repo.Create(ctx, inv)
	require.NoError(t, err)

	tx := &Transaction{
		ID:             "test-tx-1",
		InventoryID:    inv.ID,
		Type:           "test",
		Quantity:       10,
		ReferenceID:    "ref-1",
		ReferenceType:  "test",
		Notes:          "Test transaction",
	}

	err = repo.CreateTransaction(ctx, tx)
	require.NoError(t, err)

	transactions, total, err := repo.GetTransactions(ctx, inv.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, transactions, 1)
	assert.Equal(t, "test", transactions[0].Type)
	assert.Equal(t, 10, transactions[0].Quantity)
}

func TestInventoryRepository_ListLowStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)

	for i := 0; i < 3; i++ {
		product := &products.Product{
			ID:          "low-stock-product-" + string(rune('0'+i)),
			SKU:         "LOW-" + string(rune('0'+i)),
			Name:        "Low Stock Product " + string(rune('0'+i)),
			Description: "Test",
			CategoryID:  "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
			Price:       10.00,
			Cost:        5.00,
			Currency:    "USD",
			Active:      true,
		}
		err := productRepo.Create(context.Background(), product)
		require.NoError(t, err)

		inv := &Inventory{
			ID:               "low-stock-inv-" + string(rune('0'+i)),
			ProductID:        product.ID,
			WarehouseID:      "default",
			AvailableQty:     5,
			ReservedQty:      0,
			DamagedQty:       0,
			ReorderThreshold: 10,
			Version:          1,
		}
		err = NewRepository(db).Create(context.Background(), inv)
		require.NoError(t, err)
	}

	repo := NewRepository(db)
	ctx := context.Background()

	inventories, total, err := repo.List(ctx, Filter{LowStockOnly: true}, 1, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 3)
	for _, inv := range inventories {
		assert.LessOrEqual(t, inv.AvailableQty, inv.ReorderThreshold)
	}
}