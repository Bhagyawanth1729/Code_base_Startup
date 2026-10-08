package orders

import (
	"context"
	"errors"

	"github.com/forgestock/forgestock/internal/platform/database"
)

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)

type Order struct {
	ID          string      `json:"id"`
	CustomerID  string      `json:"customer_id"`
	Status      string      `json:"status"`
	Subtotal    float64     `json:"subtotal"`
	Tax         float64     `json:"tax"`
	Discount    float64     `json:"discount"`
	Total       float64     `json:"total"`
	Currency    string      `json:"currency"`
	Notes       string      `json:"notes"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	Items       []OrderItem `json:"items,omitempty"`
}

type OrderItem struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	ProductID string  `json:"product_id"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Total     float64 `json:"total"`
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, order *Order) error {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO orders (id, customer_id, status, subtotal, tax, discount, total, currency, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`
	_, err = tx.Exec(ctx, query,
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

	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Order, error) {
	query := `
		SELECT id, customer_id, status, subtotal, tax, discount, total, currency, notes, created_at, updated_at
		FROM orders WHERE id = $1
	`
	order := &Order{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&order.ID, &order.CustomerID, &order.Status, &order.Subtotal, &order.Tax, &order.Discount, &order.Total, &order.Currency, &order.Notes, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	items, err := r.GetItemsByOrderID(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Items = items

	return order, nil
}

func (r *Repository) GetItemsByOrderID(ctx context.Context, orderID string) ([]OrderItem, error) {
	query := `
		SELECT id, order_id, product_id, sku, name, quantity, price, total
		FROM order_items WHERE order_id = $1
	`
	rows, err := r.db.Pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		item := OrderItem{}
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.SKU, &item.Name, &item.Quantity, &item.Price, &item.Total); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.db.Pool.Exec(ctx, query, status, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context, filter Filter, page, pageSize int) ([]*Order, int, error) {
	offset := (page - 1) * pageSize

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if filter.CustomerID != "" {
		whereClause += " AND customer_id = $" + string(rune(argIndex+'0'))
		args = append(args, filter.CustomerID)
		argIndex++
	}
	if filter.Status != "" {
		whereClause += " AND status = $" + string(rune(argIndex+'0'))
		args = append(args, filter.Status)
		argIndex++
	}

	countQuery := "SELECT COUNT(*) FROM orders " + whereClause
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, customer_id, status, subtotal, tax, discount, total, currency, notes, created_at, updated_at
		FROM orders ` + whereClause + ` ORDER BY created_at DESC LIMIT $` + string(rune(argIndex+'0')) + ` OFFSET $` + string(rune(argIndex+1+'0'))
	args = append(args, pageSize, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orders []*Order
	for rows.Next() {
		order := &Order{}
		if err := rows.Scan(&order.ID, &order.CustomerID, &order.Status, &order.Subtotal, &order.Tax, &order.Discount, &order.Total, &order.Currency, &order.Notes, &order.CreatedAt, &order.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}

	return orders, total, nil
}

func (r *Repository) CreateAuditLog(ctx context.Context, orderID, action, actorID string, metadata map[string]interface{}) error {
	// Implementation will use the audit service
	return nil
}

type Filter struct {
	CustomerID string
	Status     string
}

const (
	StatusPending    = "pending"
	StatusConfirmed  = "confirmed"
	StatusProcessing = "processing"
	StatusShipped    = "shipped"
	StatusDelivered  = "delivered"
	StatusCancelled  = "cancelled"
)

var ValidStatusTransitions = map[string][]string{
	StatusPending:    {StatusConfirmed, StatusCancelled},
	StatusConfirmed:  {StatusProcessing, StatusCancelled},
	StatusProcessing: {StatusShipped, StatusCancelled},
	StatusShipped:    {StatusDelivered, StatusCancelled},
	StatusDelivered:  {},
	StatusCancelled:  {},
}

func IsValidTransition(from, to string) bool {
	allowed, ok := ValidStatusTransitions[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}