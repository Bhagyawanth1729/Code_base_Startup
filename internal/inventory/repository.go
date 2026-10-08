package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/forgestock/forgestock/internal/platform/database"
)

var (
	ErrInventoryNotFound     = errors.New("inventory not found")
	ErrInsufficientInventory = errors.New("insufficient inventory")
	ErrNegativeInventory     = errors.New("inventory cannot be negative")
	ErrOptimisticLock        = errors.New("optimistic lock failed, retry")
)

type Inventory struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
	WarehouseID     string `json:"warehouse_id"`
	AvailableQty    int    `json:"available_qty"`
	ReservedQty     int    `json:"reserved_qty"`
	DamagedQty      int    `json:"damaged_qty"`
	ReorderThreshold int   `json:"reorder_threshold"`
	Version         int    `json:"version"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type Transaction struct {
	ID             string `json:"id"`
	InventoryID    string `json:"inventory_id"`
	Type           string `json:"type"`
	Quantity       int    `json:"quantity"`
	ReferenceID    string `json:"reference_id"`
	ReferenceType  string `json:"reference_type"`
	Notes          string `json:"notes"`
	CreatedAt      string `json:"created_at"`
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, inv *Inventory) error {
	query := `
		INSERT INTO inventory (id, product_id, warehouse_id, available_qty, reserved_qty, damaged_qty, reorder_threshold, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, NOW(), NOW())
	`
	_, err := r.db.Pool.Exec(ctx, query,
		inv.ID, inv.ProductID, inv.WarehouseID, inv.AvailableQty, inv.ReservedQty, inv.DamagedQty, inv.ReorderThreshold)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Inventory, error) {
	query := `
		SELECT id, product_id, warehouse_id, available_qty, reserved_qty, damaged_qty, reorder_threshold, version, created_at, updated_at
		FROM inventory WHERE id = $1
	`
	inv := &Inventory{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.ProductID, &inv.WarehouseID, &inv.AvailableQty, &inv.ReservedQty,
		&inv.DamagedQty, &inv.ReorderThreshold, &inv.Version, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, ErrInventoryNotFound
	}
	return inv, nil
}

func (r *Repository) GetByProductAndWarehouse(ctx context.Context, productID, warehouseID string) (*Inventory, error) {
	query := `
		SELECT id, product_id, warehouse_id, available_qty, reserved_qty, damaged_qty, reorder_threshold, version, created_at, updated_at
		FROM inventory WHERE product_id = $1 AND warehouse_id = $2
	`
	inv := &Inventory{}
	err := r.db.Pool.QueryRow(ctx, query, productID, warehouseID).Scan(
		&inv.ID, &inv.ProductID, &inv.WarehouseID, &inv.AvailableQty, &inv.ReservedQty,
		&inv.DamagedQty, &inv.ReorderThreshold, &inv.Version, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, ErrInventoryNotFound
	}
	return inv, nil
}

func (r *Repository) Update(ctx context.Context, inv *Inventory) error {
	query := `
		UPDATE inventory SET available_qty = $1, reserved_qty = $2, damaged_qty = $3, 
		reorder_threshold = $4, version = version + 1, updated_at = NOW()
		WHERE id = $5 AND version = $6
	`
	result, err := r.db.Pool.Exec(ctx, query,
		inv.AvailableQty, inv.ReservedQty, inv.DamagedQty, inv.ReorderThreshold, inv.ID, inv.Version)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOptimisticLock
	}
	inv.Version++
	return nil
}

func (r *Repository) ReserveStock(ctx context.Context, inventoryID string, qty int, expectedVersion int) error {
	query := `
		UPDATE inventory SET available_qty = available_qty - $1, reserved_qty = reserved_qty + $1, 
		version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3 AND available_qty >= $1
	`
	result, err := r.db.Pool.Exec(ctx, query, qty, inventoryID, expectedVersion)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOptimisticLock
	}
	return nil
}

func (r *Repository) ReleaseStock(ctx context.Context, inventoryID string, qty int, expectedVersion int) error {
	query := `
		UPDATE inventory SET available_qty = available_qty + $1, reserved_qty = reserved_qty - $1, 
		version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3 AND reserved_qty >= $1
	`
	result, err := r.db.Pool.Exec(ctx, query, qty, inventoryID, expectedVersion)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOptimisticLock
	}
	return nil
}

func (r *Repository) DeductStock(ctx context.Context, inventoryID string, qty int, expectedVersion int) error {
	query := `
		UPDATE inventory SET reserved_qty = reserved_qty - $1, 
		version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3 AND reserved_qty >= $1
	`
	result, err := r.db.Pool.Exec(ctx, query, qty, inventoryID, expectedVersion)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOptimisticLock
	}
	return nil
}

func (r *Repository) AddStock(ctx context.Context, inventoryID string, qty int, expectedVersion int) error {
	query := `
		UPDATE inventory SET available_qty = available_qty + $1, 
		version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3
	`
	result, err := r.db.Pool.Exec(ctx, query, qty, inventoryID, expectedVersion)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOptimisticLock
	}
	return nil
}

func (r *Repository) AdjustStock(ctx context.Context, inventoryID string, availableDelta, reservedDelta, damagedDelta int, expectedVersion int) error {
	query := `
		UPDATE inventory SET 
			available_qty = available_qty + $1,
			reserved_qty = reserved_qty + $2,
			damaged_qty = damaged_qty + $3,
			version = version + 1, updated_at = NOW()
		WHERE id = $4 AND version = $5
		AND available_qty + $1 >= 0
		AND reserved_qty + $2 >= 0
		AND damaged_qty + $3 >= 0
	`
	result, err := r.db.Pool.Exec(ctx, query, availableDelta, reservedDelta, damagedDelta, inventoryID, expectedVersion)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOptimisticLock
	}
	return nil
}

func (r *Repository) CreateTransaction(ctx context.Context, tx *Transaction) error {
	query := `
		INSERT INTO inventory_transactions (id, inventory_id, type, quantity, reference_id, reference_type, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	_, err := r.db.Pool.Exec(ctx, query,
		tx.ID, tx.InventoryID, tx.Type, tx.Quantity, tx.ReferenceID, tx.ReferenceType, tx.Notes)
	return err
}

func (r *Repository) List(ctx context.Context, filter Filter, page, pageSize int) ([]*Inventory, int, error) {
	offset := (page - 1) * pageSize

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.ProductID != "" {
		whereClause += " AND product_id = $" + string(rune(argIndex+'0'))
		args = append(args, filter.ProductID)
		argIndex++
	}
	if filter.WarehouseID != "" {
		whereClause += " AND warehouse_id = $" + string(rune(argIndex+'0'))
		args = append(args, filter.WarehouseID)
		argIndex++
	}
	if filter.LowStockOnly {
		whereClause += " AND available_qty <= reorder_threshold"
	}

	countQuery := "SELECT COUNT(*) FROM inventory " + whereClause
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, product_id, warehouse_id, available_qty, reserved_qty, damaged_qty, reorder_threshold, version, created_at, updated_at
		FROM inventory ` + whereClause + ` ORDER BY created_at DESC LIMIT $` + string(rune(argIndex+'0')) + ` OFFSET $` + string(rune(argIndex+1+'0'))
	args = append(args, pageSize, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var inventories []*Inventory
	for rows.Next() {
		inv := &Inventory{}
		if err := rows.Scan(
			&inv.ID, &inv.ProductID, &inv.WarehouseID, &inv.AvailableQty, &inv.ReservedQty,
			&inv.DamagedQty, &inv.ReorderThreshold, &inv.Version, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, 0, err
		}
		inventories = append(inventories, inv)
	}

	return inventories, total, nil
}

func (r *Repository) GetTransactions(ctx context.Context, inventoryID string, page, pageSize int) ([]*Transaction, int, error) {
	offset := (page - 1) * pageSize

	countQuery := `SELECT COUNT(*) FROM inventory_transactions WHERE inventory_id = $1`
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, inventoryID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, inventory_id, type, quantity, reference_id, reference_type, notes, created_at
		FROM inventory_transactions WHERE inventory_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Pool.Query(ctx, query, inventoryID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var transactions []*Transaction
	for rows.Next() {
		tx := &Transaction{}
		if err := rows.Scan(&tx.ID, &tx.InventoryID, &tx.Type, &tx.Quantity, &tx.ReferenceID, &tx.ReferenceType, &tx.Notes, &tx.CreatedAt); err != nil {
			return nil, 0, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, total, nil
}

type Filter struct {
	ProductID      string
	WarehouseID    string
	LowStockOnly   bool
}