package orders

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

func createTestUser(t *testing.T, db *database.DB) string {
	// Just use a fixed ID for testing
	return "44444444-4444-4444-4444-444444444444"
}

func createTestProduct(t *testing.T, repo *products.Repository) *products.Product {
	product := &products.Product{
		ID:          "order-test-product-1",
		SKU:         "ORD-001",
		Name:        "Order Test Product",
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

func TestOrderRepository_CreateAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	order := &Order{
		ID:         "test-order-1",
		CustomerID: "44444444-4444-4444-4444-444444444444",
		Status:     StatusPending,
		Subtotal:   99.99,
		Tax:        10.00,
		Discount:   0.00,
		Total:      109.99,
		Currency:   "USD",
		Notes:      "Test order",
		Items: []OrderItem{
			{
				ID:        "test-item-1",
				OrderID:   "test-order-1",
				ProductID: product.ID,
				SKU:       product.SKU,
				Name:      product.Name,
				Quantity:  1,
				Price:     product.Price,
				Total:     product.Price,
			},
		},
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, found.ID)
	assert.Equal(t, order.CustomerID, found.CustomerID)
	assert.Equal(t, order.Status, found.Status)
	assert.Equal(t, order.Subtotal, found.Subtotal)
	assert.Equal(t, order.Tax, found.Tax)
	assert.Equal(t, order.Total, found.Total)
	assert.Len(t, found.Items, 1)
	assert.Equal(t, order.Items[0].ProductID, found.Items[0].ProductID)
	assert.Equal(t, order.Items[0].Quantity, found.Items[0].Quantity)
}

func TestOrderRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	order := &Order{
		ID:         "test-order-2",
		CustomerID: "44444444-4444-4444-4444-444444444444",
		Status:     StatusPending,
		Subtotal:   99.99,
		Tax:        10.00,
		Discount:   0.00,
		Total:      109.99,
		Currency:   "USD",
		Notes:      "Test order",
		Items: []OrderItem{
			{
				ID:        "test-item-2",
				OrderID:   "test-order-2",
				ProductID: product.ID,
				SKU:       product.SKU,
				Name:      product.Name,
				Quantity:  1,
				Price:     product.Price,
				Total:     product.Price,
			},
		},
	}

	err := repo.Create(ctx, order)
	require.NoError(t, err)

	err = repo.UpdateStatus(ctx, order.ID, StatusConfirmed)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, order.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusConfirmed, found.Status)
}

func TestOrderRepository_ListWithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	productRepo := products.NewRepository(db)
	product := createTestProduct(t, productRepo)

	repo := NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		order := &Order{
			ID:         "list-order-" + string(rune('0'+i)),
			CustomerID: "44444444-4444-4444-4444-444444444444",
			Status:     StatusPending,
			Subtotal:   50.00 + float64(i)*10,
			Tax:        5.00 + float64(i),
			Discount:   0.00,
			Total:      55.00 + float64(i)*11,
			Currency:   "USD",
			Notes:      "",
			Items: []OrderItem{
				{
					ID:        "list-item-" + string(rune('0'+i)),
					OrderID:   "list-order-" + string(rune('0'+i)),
					ProductID: product.ID,
					SKU:       product.SKU,
					Name:      product.Name,
					Quantity:  1,
					Price:     product.Price,
					Total:     product.Price,
				},
			},
		}
		err := repo.Create(ctx, order)
		require.NoError(t, err)
	}

	filter := Filter{
		CustomerID: "44444444-4444-4444-4444-444444444444",
		Status:     StatusPending,
	}
	orders, total, err := repo.List(ctx, filter, 1, 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 3)
	assert.LessOrEqual(t, len(orders), 10)
}

func TestOrderRepository_ValidStatusTransitions(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected bool
	}{
		{StatusPending, StatusConfirmed, true},
		{StatusPending, StatusCancelled, true},
		{StatusPending, StatusProcessing, false},
		{StatusConfirmed, StatusProcessing, true},
		{StatusConfirmed, StatusCancelled, true},
		{StatusConfirmed, StatusShipped, false},
		{StatusProcessing, StatusShipped, true},
		{StatusProcessing, StatusCancelled, true},
		{StatusProcessing, StatusDelivered, false},
		{StatusShipped, StatusDelivered, true},
		{StatusShipped, StatusCancelled, true},
		{StatusDelivered, StatusCancelled, false},
		{StatusCancelled, StatusPending, false},
	}

	for _, tt := range tests {
		result := IsValidTransition(tt.from, tt.to)
		assert.Equal(t, tt.expected, result, "transition from %s to %s", tt.from, tt.to)
	}
}