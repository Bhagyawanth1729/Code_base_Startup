package payments

import (
	"context"
	"errors"
)

var (
	ErrPaymentNotPending = errors.New("payment is not in pending state")
	ErrPaymentNotAuthorized = errors.New("payment is not authorized")
)

type Provider interface {
	Authorize(ctx context.Context, payment *Payment) error
	Capture(ctx context.Context, payment *Payment) error
	Refund(ctx context.Context, payment *Payment) error
	Void(ctx context.Context, payment *Payment) error
}

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (m *MockProvider) Authorize(ctx context.Context, payment *Payment) error {
	if payment.Status != StatusPending {
		return ErrPaymentNotPending
	}
	payment.Status = StatusAuthorized
	payment.ProviderRef = "mock_auth_" + payment.ID
	return nil
}

func (m *MockProvider) Capture(ctx context.Context, payment *Payment) error {
	if payment.Status != StatusAuthorized {
		return ErrPaymentNotAuthorized
	}
	payment.Status = StatusCaptured
	payment.ProviderRef = "mock_capture_" + payment.ID
	return nil
}

func (m *MockProvider) Refund(ctx context.Context, payment *Payment) error {
	if payment.Status != StatusCaptured && payment.Status != StatusAuthorized {
		return errors.New("payment cannot be refunded")
	}
	payment.Status = StatusRefunded
	payment.ProviderRef = "mock_refund_" + payment.ID
	return nil
}

func (m *MockProvider) Void(ctx context.Context, payment *Payment) error {
	if payment.Status != StatusPending && payment.Status != StatusAuthorized {
		return errors.New("payment cannot be voided")
	}
	payment.Status = StatusVoided
	payment.ProviderRef = "mock_void_" + payment.ID
	return nil
}

const (
	StatusPending    = "pending"
	StatusAuthorized = "authorized"
	StatusCaptured   = "captured"
	StatusFailed     = "failed"
	StatusRefunded   = "refunded"
	StatusVoided     = "voided"
)