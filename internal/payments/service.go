package payments

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
)

type Service struct {
	repo     *Repository
	provider Provider
	db       *database.DB
	log      *logger.Logger
}

func NewService(repo *Repository, provider Provider, db *database.DB, log *logger.Logger) *Service {
	return &Service{repo: repo, provider: provider, db: db, log: log}
}

func (s *Service) CreatePayment(ctx context.Context, orderID string, amount float64, currency string) (*Payment, error) {
	payment := &Payment{
		ID:        uuid.New().String(),
		OrderID:   orderID,
		Amount:    amount,
		Currency:  currency,
		Status:    StatusPending,
		Provider:  "mock",
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "payment_created", "payment", payment.ID, map[string]interface{}{
		"order_id": payment.OrderID,
		"amount":   payment.Amount,
	})

	return payment, nil
}

func (s *Service) AuthorizePayment(ctx context.Context, paymentID string) (*Payment, error) {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if err := s.provider.Authorize(ctx, payment); err != nil {
		payment.Status = StatusFailed
		payment.FailureReason = err.Error()
		s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, payment.FailureReason)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, ""); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "payment_authorized", "payment", payment.ID, map[string]interface{}{
		"order_id": payment.OrderID,
		"amount":   payment.Amount,
	})

	return payment, nil
}

func (s *Service) CapturePayment(ctx context.Context, paymentID string) (*Payment, error) {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if err := s.provider.Capture(ctx, payment); err != nil {
		payment.Status = StatusFailed
		payment.FailureReason = err.Error()
		s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, payment.FailureReason)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, ""); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "payment_captured", "payment", payment.ID, map[string]interface{}{
		"order_id": payment.OrderID,
		"amount":   payment.Amount,
	})

	return payment, nil
}

func (s *Service) RefundPayment(ctx context.Context, paymentID string) (*Payment, error) {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if err := s.provider.Refund(ctx, payment); err != nil {
		payment.FailureReason = err.Error()
		s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, payment.FailureReason)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, ""); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "payment_refunded", "payment", payment.ID, map[string]interface{}{
		"order_id": payment.OrderID,
		"amount":   payment.Amount,
	})

	return payment, nil
}

func (s *Service) VoidPayment(ctx context.Context, paymentID string) (*Payment, error) {
	payment, err := s.repo.GetByID(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	if err := s.provider.Void(ctx, payment); err != nil {
		payment.FailureReason = err.Error()
		s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, payment.FailureReason)
		return nil, err
	}

	if err := s.repo.UpdateStatus(ctx, payment.ID, payment.Status, payment.ProviderRef, ""); err != nil {
		return nil, err
	}

	s.log.AuditLog(ctx, "payment_voided", "payment", payment.ID, map[string]interface{}{
		"order_id": payment.OrderID,
		"amount":   payment.Amount,
	})

	return payment, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (*Payment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByOrderID(ctx context.Context, orderID string) (*Payment, error) {
	return s.repo.GetByOrderID(ctx, orderID)
}

func (s *Service) Refund(ctx context.Context, paymentID string) error {
	_, err := s.RefundPayment(ctx, paymentID)
	return err
}