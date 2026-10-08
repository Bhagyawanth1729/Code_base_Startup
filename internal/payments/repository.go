package payments

import (
	"context"
	"errors"

	"github.com/forgestock/forgestock/internal/platform/database"
)

var (
	ErrPaymentNotFound = errors.New("payment not found")
)

type Payment struct {
	ID            string  `json:"id"`
	OrderID       string  `json:"order_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	Provider      string  `json:"provider"`
	ProviderRef   string  `json:"provider_ref"`
	FailureReason string  `json:"failure_reason"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, payment *Payment) error {
	query := `
		INSERT INTO payments (id, order_id, amount, currency, status, provider, provider_ref, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`
	_, err := r.db.Pool.Exec(ctx, query,
		payment.ID, payment.OrderID, payment.Amount, payment.Currency, payment.Status, payment.Provider, payment.ProviderRef)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, provider, provider_ref, failure_reason, created_at, updated_at
		FROM payments WHERE id = $1
	`
	payment := &Payment{}
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&payment.ID, &payment.OrderID, &payment.Amount, &payment.Currency, &payment.Status,
		&payment.Provider, &payment.ProviderRef, &payment.FailureReason, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return nil, ErrPaymentNotFound
	}
	return payment, nil
}

func (r *Repository) GetByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	query := `
		SELECT id, order_id, amount, currency, status, provider, provider_ref, failure_reason, created_at, updated_at
		FROM payments WHERE order_id = $1
	`
	payment := &Payment{}
	err := r.db.Pool.QueryRow(ctx, query, orderID).Scan(
		&payment.ID, &payment.OrderID, &payment.Amount, &payment.Currency, &payment.Status,
		&payment.Provider, &payment.ProviderRef, &payment.FailureReason, &payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		return nil, ErrPaymentNotFound
	}
	return payment, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status, providerRef, failureReason string) error {
	query := `UPDATE payments SET status = $1, provider_ref = $2, failure_reason = $3, updated_at = NOW() WHERE id = $4`
	result, err := r.db.Pool.Exec(ctx, query, status, providerRef, failureReason, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrPaymentNotFound
	}
	return nil
}