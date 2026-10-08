package inventory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/forgestock/forgestock/internal/products"
)

type Service struct {
	repo       *Repository
	productRepo *products.Repository
	db         *database.DB
	log        *logger.Logger
}

func NewService(repo *Repository, productRepo *products.Repository, db *database.DB, log *logger.Logger) *Service {
	return &Service{repo: repo, productRepo: productRepo, db: db, log: log}
}

type CreateInventoryRequest struct {
	ProductID        string `json:"product_id" validate:"required"`
	WarehouseID      string `json:"warehouse_id" validate:"required"`
	AvailableQty     int    `json:"available_qty" validate:"gte=0"`
	ReservedQty      int    `json:"reserved_qty" validate:"gte=0"`
	DamagedQty       int    `json:"damaged_qty" validate:"gte=0"`
	ReorderThreshold int    `json:"reorder_threshold" validate:"gte=0"`
}

type ReserveStockRequest struct {
	ProductID    string `json:"product_id" validate:"required"`
	WarehouseID  string `json:"warehouse_id" validate:"required"`
	Quantity     int    `json:"quantity" validate:"required,gt=0"`
	ReferenceID  string `json:"reference_id" validate:"required"`
	ReferenceType string `json:"reference_type" validate:"required"`
	Notes        string `json:"notes"`
}

type ReleaseStockRequest struct {
	ProductID     string `json:"product_id" validate:"required"`
	WarehouseID   string `json:"warehouse_id" validate:"required"`
	Quantity      int    `json:"quantity" validate:"required,gt=0"`
	ReferenceID   string `json:"reference_id" validate:"required"`
	ReferenceType string `json:"reference_type" validate:"required"`
	Notes         string `json:"notes"`
}

type AdjustStockRequest struct {
	ProductID       string `json:"product_id" validate:"required"`
	WarehouseID     string `json:"warehouse_id" validate:"required"`
	AvailableDelta  int    `json:"available_delta"`
	ReservedDelta   int    `json:"reserved_delta"`
	DamagedDelta    int    `json:"damaged_delta"`
	Notes           string `json:"notes" validate:"required"`
}

type AddStockRequest struct {
	ProductID     string `json:"product_id" validate:"required"`
	WarehouseID   string `json:"warehouse_id" validate:"required"`
	Quantity      int    `json:"quantity" validate:"required,gt=0"`
	ReferenceID   string `json:"reference_id" validate:"required"`
	ReferenceType string `json:"reference_type" validate:"required"`
	Notes         string `json:"notes"`
}

func (s *Service) Create(ctx context.Context, req CreateInventoryRequest) (*Inventory, error) {
	_, err := s.productRepo.GetByID(ctx, req.ProductID)
	if err != nil {
		return nil, errors.New("invalid product")
	}

	existing, err := s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("inventory already exists for product %s in warehouse %s", req.ProductID, req.WarehouseID)
	}

	inv := &Inventory{
		ID:               uuid.New().String(),
		ProductID:        req.ProductID,
		WarehouseID:      req.WarehouseID,
		AvailableQty:     req.AvailableQty,
		ReservedQty:      req.ReservedQty,
		DamagedQty:       req.DamagedQty,
		ReorderThreshold: req.ReorderThreshold,
		Version:          1,
	}

	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, err
	}

	txRecord := &Transaction{
		ID:             uuid.New().String(),
		InventoryID:    inv.ID,
		Type:           "initial",
		Quantity:       req.AvailableQty,
		ReferenceID:    "",
		ReferenceType:  "initial",
		Notes:          "Initial inventory creation",
	}
	if err := s.repo.CreateTransaction(ctx, txRecord); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "inventory_created", "inventory", inv.ID, map[string]interface{}{
		"product_id":        inv.ProductID,
		"warehouse_id":      inv.WarehouseID,
		"available_qty":     inv.AvailableQty,
		"reorder_threshold": inv.ReorderThreshold,
	})

	return inv, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Inventory, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByProductAndWarehouse(ctx context.Context, productID, warehouseID string) (*Inventory, error) {
	return s.repo.GetByProductAndWarehouse(ctx, productID, warehouseID)
}

func (s *Service) ReserveStock(ctx context.Context, req ReserveStockRequest) (*Inventory, error) {
	inv, err := s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
	if err != nil {
		return nil, err
	}

	if inv.AvailableQty < req.Quantity {
		return nil, ErrInsufficientInventory
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.repo.ReserveStock(ctx, inv.ID, req.Quantity, inv.Version)
		if err == nil {
			break
		}
		if errors.Is(err, ErrOptimisticLock) {
			inv, err = s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
			if err != nil {
				return nil, err
			}
			if inv.AvailableQty < req.Quantity {
				return nil, ErrInsufficientInventory
			}
			continue
		}
		return nil, err
	}

	if err != nil {
		return nil, ErrOptimisticLock
	}

	txRecord := &Transaction{
		ID:             uuid.New().String(),
		InventoryID:    inv.ID,
		Type:           "reserve",
		Quantity:       req.Quantity,
		ReferenceID:    req.ReferenceID,
		ReferenceType:  req.ReferenceType,
		Notes:          req.Notes,
	}
	if err := s.repo.CreateTransaction(ctx, txRecord); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "stock_reserved", "inventory", inv.ID, map[string]interface{}{
		"quantity":       req.Quantity,
		"reference_id":   req.ReferenceID,
		"reference_type": req.ReferenceType,
	})

	inv, _ = s.repo.GetByID(ctx, inv.ID)
	return inv, nil
}

func (s *Service) ReleaseStock(ctx context.Context, req ReleaseStockRequest) (*Inventory, error) {
	inv, err := s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
	if err != nil {
		return nil, err
	}

	if inv.ReservedQty < req.Quantity {
		return nil, errors.New("insufficient reserved stock to release")
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.repo.ReleaseStock(ctx, inv.ID, req.Quantity, inv.Version)
		if err == nil {
			break
		}
		if errors.Is(err, ErrOptimisticLock) {
			inv, err = s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
			if err != nil {
				return nil, err
			}
			if inv.ReservedQty < req.Quantity {
				return nil, errors.New("insufficient reserved stock to release")
			}
			continue
		}
		return nil, err
	}

	if err != nil {
		return nil, ErrOptimisticLock
	}

	txRecord := &Transaction{
		ID:             uuid.New().String(),
		InventoryID:    inv.ID,
		Type:           "release",
		Quantity:       req.Quantity,
		ReferenceID:    req.ReferenceID,
		ReferenceType:  req.ReferenceType,
		Notes:          req.Notes,
	}
	if err := s.repo.CreateTransaction(ctx, txRecord); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "stock_released", "inventory", inv.ID, map[string]interface{}{
		"quantity":       req.Quantity,
		"reference_id":   req.ReferenceID,
		"reference_type": req.ReferenceType,
	})

	inv, _ = s.repo.GetByID(ctx, inv.ID)
	return inv, nil
}

func (s *Service) DeductStock(ctx context.Context, productID, warehouseID string, qty int, referenceID, referenceType, notes string) (*Inventory, error) {
	inv, err := s.repo.GetByProductAndWarehouse(ctx, productID, warehouseID)
	if err != nil {
		return nil, err
	}

	if inv.ReservedQty < qty {
		return nil, errors.New("insufficient reserved stock to deduct")
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.repo.DeductStock(ctx, inv.ID, qty, inv.Version)
		if err == nil {
			break
		}
		if errors.Is(err, ErrOptimisticLock) {
			inv, err = s.repo.GetByProductAndWarehouse(ctx, productID, warehouseID)
			if err != nil {
				return nil, err
			}
			if inv.ReservedQty < qty {
				return nil, errors.New("insufficient reserved stock to deduct")
			}
			continue
		}
		return nil, err
	}

	if err != nil {
		return nil, ErrOptimisticLock
	}

	txRecord := &Transaction{
		ID:             uuid.New().String(),
		InventoryID:    inv.ID,
		Type:           "deduct",
		Quantity:       qty,
		ReferenceID:    referenceID,
		ReferenceType:  referenceType,
		Notes:          notes,
	}
	if err := s.repo.CreateTransaction(ctx, txRecord); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "stock_deducted", "inventory", inv.ID, map[string]interface{}{
		"quantity":       qty,
		"reference_id":   referenceID,
		"reference_type": referenceType,
	})

	inv, _ = s.repo.GetByID(ctx, inv.ID)
	return inv, nil
}

func (s *Service) AddStock(ctx context.Context, req AddStockRequest) (*Inventory, error) {
	inv, err := s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
	if err != nil {
		if errors.Is(err, ErrInventoryNotFound) {
			return s.Create(ctx, CreateInventoryRequest{
				ProductID:   req.ProductID,
				WarehouseID: req.WarehouseID,
				AvailableQty: req.Quantity,
			})
		}
		return nil, err
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.repo.AddStock(ctx, inv.ID, req.Quantity, inv.Version)
		if err == nil {
			break
		}
		if errors.Is(err, ErrOptimisticLock) {
			inv, err = s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
			if err != nil {
				return nil, err
			}
			continue
		}
		return nil, err
	}

	if err != nil {
		return nil, ErrOptimisticLock
	}

	txRecord := &Transaction{
		ID:             uuid.New().String(),
		InventoryID:    inv.ID,
		Type:           "receive",
		Quantity:       req.Quantity,
		ReferenceID:    req.ReferenceID,
		ReferenceType:  req.ReferenceType,
		Notes:          req.Notes,
	}
	if err := s.repo.CreateTransaction(ctx, txRecord); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "stock_received", "inventory", inv.ID, map[string]interface{}{
		"quantity":       req.Quantity,
		"reference_id":   req.ReferenceID,
		"reference_type": req.ReferenceType,
	})

	inv, _ = s.repo.GetByID(ctx, inv.ID)
	return inv, nil
}

func (s *Service) AdjustStock(ctx context.Context, req AdjustStockRequest) (*Inventory, error) {
	inv, err := s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
	if err != nil {
		return nil, err
	}

	if inv.AvailableQty+req.AvailableDelta < 0 ||
		inv.ReservedQty+req.ReservedDelta < 0 ||
		inv.DamagedQty+req.DamagedDelta < 0 {
		return nil, ErrNegativeInventory
	}

	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err = s.repo.AdjustStock(ctx, inv.ID, req.AvailableDelta, req.ReservedDelta, req.DamagedDelta, inv.Version)
		if err == nil {
			break
		}
		if errors.Is(err, ErrOptimisticLock) {
			inv, err = s.repo.GetByProductAndWarehouse(ctx, req.ProductID, req.WarehouseID)
			if err != nil {
				return nil, err
			}
			if inv.AvailableQty+req.AvailableDelta < 0 ||
				inv.ReservedQty+req.ReservedDelta < 0 ||
				inv.DamagedQty+req.DamagedDelta < 0 {
				return nil, ErrNegativeInventory
			}
			continue
		}
		return nil, err
	}

	if err != nil {
		return nil, ErrOptimisticLock
	}

	txRecord := &Transaction{
		ID:             uuid.New().String(),
		InventoryID:    inv.ID,
		Type:           "adjustment",
		Quantity:       req.AvailableDelta + req.ReservedDelta + req.DamagedDelta,
		ReferenceID:    "",
		ReferenceType:  "adjustment",
		Notes:          req.Notes,
	}
	if err := s.repo.CreateTransaction(ctx, txRecord); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "stock_adjusted", "inventory", inv.ID, map[string]interface{}{
		"available_delta": req.AvailableDelta,
		"reserved_delta":  req.ReservedDelta,
		"damaged_delta":   req.DamagedDelta,
		"notes":           req.Notes,
	})

	inv, _ = s.repo.GetByID(ctx, inv.ID)
	return inv, nil
}

func (s *Service) List(ctx context.Context, filter Filter, page, pageSize int) ([]*Inventory, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, filter, page, pageSize)
}

func (s *Service) GetTransactions(ctx context.Context, inventoryID string, page, pageSize int) ([]*Transaction, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.GetTransactions(ctx, inventoryID, page, pageSize)
}