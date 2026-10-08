package orders

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/inventory"
	"github.com/forgestock/forgestock/internal/payments"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/forgestock/forgestock/internal/products"
)

type Service struct {
	repo         *Repository
	productRepo  *products.Repository
	inventoryRepo *inventory.Repository
	inventorySvc *inventory.Service
	paymentSvc   *payments.Service
	db           *database.DB
	log          *logger.Logger
}

func NewService(
	repo *Repository,
	productRepo *products.Repository,
	inventoryRepo *inventory.Repository,
	inventorySvc *inventory.Service,
	paymentSvc *payments.Service,
	db *database.DB,
	log *logger.Logger,
) *Service {
	return &Service{
		repo:          repo,
		productRepo:   productRepo,
		inventoryRepo: inventoryRepo,
		inventorySvc:  inventorySvc,
		paymentSvc:    paymentSvc,
		db:            db,
		log:           log,
	}
}

type CreateOrderRequest struct {
	CustomerID string              `json:"customer_id" validate:"required"`
	Items      []CreateOrderItem   `json:"items" validate:"required,min=1,dive"`
	Notes      string              `json:"notes"`
	Currency   string              `json:"currency" validate:"required,len=3"`
}

type CreateOrderItem struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required,gt=0"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=pending confirmed processing shipped delivered cancelled"`
}

func (s *Service) Create(ctx context.Context, req CreateOrderRequest) (*Order, error) {
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var subtotal float64
	orderItems := make([]OrderItem, 0, len(req.Items))
	reservedInventories := make([]*inventory.Inventory, 0, len(req.Items))

	for _, itemReq := range req.Items {
		product, err := s.productRepo.GetByID(ctx, itemReq.ProductID)
		if err != nil {
			return nil, fmt.Errorf("product %s not found", itemReq.ProductID)
		}

		if !product.Active {
			return nil, fmt.Errorf("product %s is not active", product.SKU)
		}

		inv, err := s.inventoryRepo.GetByProductAndWarehouse(ctx, product.ID, "default")
		if err != nil {
			return nil, fmt.Errorf("inventory not found for product %s", product.SKU)
		}

		if inv.AvailableQty < itemReq.Quantity {
			return nil, fmt.Errorf("insufficient inventory for product %s: available %d, requested %d", product.SKU, inv.AvailableQty, itemReq.Quantity)
		}

		itemTotal := product.Price * float64(itemReq.Quantity)
		subtotal += itemTotal

		orderItem := OrderItem{
			ID:        uuid.New().String(),
			ProductID: product.ID,
			SKU:       product.SKU,
			Name:      product.Name,
			Quantity:  itemReq.Quantity,
			Price:     product.Price,
			Total:     itemTotal,
		}
		orderItems = append(orderItems, orderItem)
		reservedInventories = append(reservedInventories, inv)
	}

	tax := subtotal * 0.1
	discount := 0.0
	total := subtotal + tax - discount

	order := &Order{
		ID:         uuid.New().String(),
		CustomerID: req.CustomerID,
		Status:     StatusPending,
		Subtotal:   subtotal,
		Tax:        tax,
		Discount:   discount,
		Total:      total,
		Currency:   req.Currency,
		Notes:      req.Notes,
		Items:      orderItems,
	}

	if err := s.createOrderInTx(ctx, tx, order); err != nil {
		return nil, err
	}

	for i, inv := range reservedInventories {
		item := orderItems[i]
		if err := s.inventorySvc.ReserveStock(ctx, inventory.ReserveStockRequest{
			ProductID:     item.ProductID,
			WarehouseID:   inv.WarehouseID,
			Quantity:      item.Quantity,
			ReferenceID:   order.ID,
			ReferenceType: "order",
			Notes:         "Reserved for order " + order.ID,
		}); err != nil {
			return nil, fmt.Errorf("failed to reserve stock for %s: %w", item.SKU, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "order_created", "order", order.ID, map[string]interface{}{
		"customer_id": order.CustomerID,
		"total":       order.Total,
		"items_count": len(order.Items),
	})

	return order, nil
}

func (s *Service) createOrderInTx(ctx context.Context, tx database.Tx, order *Order) error {
	query := `
		INSERT INTO orders (id, customer_id, status, subtotal, tax, discount, total, currency, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err := tx.Exec(ctx, query,
		order.ID, order.CustomerID, order.Status, order.Subtotal, order.Tax, order.Discount, order.Total, order.Currency, order.Notes)
	if err != nil {
		return err
	}

	for _, item := range order.Items {
		itemQuery := `
			INSERT INTO order_items (id, order_id, product_id, sku, name, quantity, price, total)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err = tx.Exec(ctx, itemQuery,
			item.ID, order.ID, item.ProductID, item.SKU, item.Name, item.Quantity, item.Price, item.Total)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateStatus(ctx context.Context, id, status string) error {
	order, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !IsValidTransition(order.Status, status) {
		return ErrInvalidStatusTransition
	}

	if status == StatusCancelled && order.Status != StatusCancelled {
		for _, item := range order.Items {
			inv, err := s.inventoryRepo.GetByProductAndWarehouse(ctx, item.ProductID, "default")
			if err == nil {
				s.inventorySvc.ReleaseStock(ctx, inventory.ReleaseStockRequest{
					ProductID:     item.ProductID,
					WarehouseID:   inv.WarehouseID,
					Quantity:      item.Quantity,
					ReferenceID:   order.ID,
					ReferenceType: "order",
					Notes:         "Released due to order cancellation",
				})
			}
		}

		if order.Status == StatusConfirmed || order.Status == StatusProcessing {
			payment, err := s.paymentSvc.GetByOrderID(ctx, order.ID)
			if err == nil && payment != nil {
				s.paymentSvc.Refund(ctx, payment.ID)
			}
		}
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	s.log.AuditLog(ctx, "order_status_changed", "order", id, map[string]interface{}{
		"from_status": order.Status,
		"to_status":   status,
	})

	return nil
}

func (s *Service) Cancel(ctx context.Context, id string) error {
	return s.UpdateStatus(ctx, id, StatusCancelled)
}

func (s *Service) List(ctx context.Context, filter Filter, page, pageSize int) ([]*Order, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, filter, page, pageSize)
}

func (s *Service) ListByCustomer(ctx context.Context, customerID string, page, pageSize int) ([]*Order, int, error) {
	filter := Filter{CustomerID: customerID}
	return s.List(ctx, filter, page, pageSize)
}