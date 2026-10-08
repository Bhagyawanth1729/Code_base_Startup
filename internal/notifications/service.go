package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/forgestock/forgestock/internal/platform/redis"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

type NotificationType string

const (
	NotificationTypeEmail       NotificationType = "email"
	NotificationTypeOrderConfirmation NotificationType = "order_confirmation"
	NotificationTypeShipment    NotificationType = "shipment_notification"
	NotificationTypeLowStock    NotificationType = "low_stock_alert"
)

type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
)

type Notification struct {
	ID        string                 `json:"id"`
	Type      NotificationType       `json:"type"`
	Recipient string                 `json:"recipient"`
	Subject   string                 `json:"subject"`
	Body      string                 `json:"body"`
	Status    NotificationStatus     `json:"status"`
	Metadata  map[string]interface{} `json:"metadata"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt string                 `json:"created_at"`
	SentAt    string                 `json:"sent_at,omitempty"`
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type ConsoleEmailSender struct{}

func NewConsoleEmailSender() *ConsoleEmailSender {
	return &ConsoleEmailSender{}
}

func (c *ConsoleEmailSender) Send(ctx context.Context, to, subject, body string) error {
	fmt.Printf("\n=== EMAIL ===\nTo: %s\nSubject: %s\nBody:\n%s\n=============\n", to, subject, body)
	return nil
}

type Service struct {
	db      *database.DB
	redis   *redis.Client
	sender  EmailSender
	log     *logger.Logger
	queue   string
}

func NewService(db *database.DB, redisClient *redis.Client, sender EmailSender, log *logger.Logger) *Service {
	return &Service{
		db:     db,
		redis:  redisClient,
		sender: sender,
		log:    log,
		queue:  "notifications",
	}
}

func (s *Service) CreateNotification(ctx context.Context, notification *Notification) error {
	if notification.ID == "" {
		notification.ID = uuid.New().String()
	}
	notification.Status = NotificationStatusPending

	query := `
		INSERT INTO notifications (id, type, recipient, subject, body, status, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	metadata, _ := json.Marshal(notification.Metadata)
	_, err := s.db.Pool.Exec(ctx, query,
		notification.ID, notification.Type, notification.Recipient, notification.Subject,
		notification.Body, notification.Status, metadata)
	if err != nil {
		return err
	}

	return s.Enqueue(ctx, notification.ID)
}

func (s *Service) Enqueue(ctx context.Context, notificationID string) error {
	return s.redis.LPush(ctx, s.queue, notificationID).Err()
}

func (s *Service) ProcessQueue(ctx context.Context) error {
	for {
		result, err := s.redis.RPop(ctx, s.queue).Result()
		if err != nil {
			if err.Error() == "redis: nil" {
				return nil
			}
			return err
		}

		if err := s.ProcessNotification(ctx, result); err != nil {
			s.log.Error().Err(err).Str("notification_id", result).Msg("Failed to process notification")
		}
	}
}

func (s *Service) ProcessNotification(ctx context.Context, notificationID string) error {
	notification, err := s.GetNotification(ctx, notificationID)
	if err != nil {
		return err
	}

	if notification.Status != NotificationStatusPending {
		return nil
	}

	err = s.sendNotification(ctx, notification)
	if err != nil {
		notification.Status = NotificationStatusFailed
		notification.Error = err.Error()
		s.updateNotification(ctx, notification)
		return err
	}

	notification.Status = NotificationStatusSent
	return s.updateNotification(ctx, notification)
}

func (s *Service) sendNotification(ctx context.Context, notification *Notification) error {
	switch notification.Type {
	case NotificationTypeEmail, NotificationTypeOrderConfirmation, NotificationTypeShipment, NotificationTypeLowStock:
		return s.sender.Send(ctx, notification.Recipient, notification.Subject, notification.Body)
	default:
		return errors.New("unknown notification type")
	}
}

func (s *Service) updateNotification(ctx context.Context, notification *Notification) error {
	query := `
		UPDATE notifications SET status = $1, error = $2, sent_at = CASE WHEN $1 = 'sent' THEN NOW() ELSE sent_at END
		WHERE id = $3
	`
	_, err := s.db.Pool.Exec(ctx, query, notification.Status, notification.Error, notification.ID)
	return err
}

func (s *Service) GetNotification(ctx context.Context, id string) (*Notification, error) {
	query := `
		SELECT id, type, recipient, subject, body, status, metadata, error, created_at, sent_at
		FROM notifications WHERE id = $1
	`
	notification := &Notification{}
	var metadata []byte
	err := s.db.Pool.QueryRow(ctx, query, id).Scan(
		&notification.ID, &notification.Type, &notification.Recipient, &notification.Subject,
		&notification.Body, &notification.Status, &metadata, &notification.Error,
		&notification.CreatedAt, &notification.SentAt)
	if err != nil {
		return nil, ErrNotificationNotFound
	}
	json.Unmarshal(metadata, &notification.Metadata)
	return notification, nil
}

func (s *Service) SendOrderConfirmation(ctx context.Context, orderID, customerEmail, customerName string, total float64, items []map[string]interface{}) error {
	subject := fmt.Sprintf("Order Confirmation - %s", orderID)
	body := fmt.Sprintf("Dear %s,\n\nYour order %s has been confirmed.\nTotal: %.2f\n\nItems:\n", customerName, orderID, total)
	for _, item := range items {
		body += fmt.Sprintf("- %s x%d @ %.2f\n", item["name"], item["quantity"], item["price"])
	}
	body += "\nThank you for your order!"

	notification := &Notification{
		Type:       NotificationTypeOrderConfirmation,
		Recipient:  customerEmail,
		Subject:    subject,
		Body:       body,
		Metadata:   map[string]interface{}{"order_id": orderID},
	}
	return s.CreateNotification(ctx, notification)
}

func (s *Service) SendShipmentNotification(ctx context.Context, orderID, customerEmail, customerName, trackingNumber string) error {
	subject := fmt.Sprintf("Your Order %s Has Shipped", orderID)
	body := fmt.Sprintf("Dear %s,\n\nYour order %s has been shipped.\nTracking Number: %s\n\nYou can track your package using this number.\n\nThank you!", customerName, orderID, trackingNumber)

	notification := &Notification{
		Type:       NotificationTypeShipment,
		Recipient:  customerEmail,
		Subject:    subject,
		Body:       body,
		Metadata:   map[string]interface{}{"order_id": orderID, "tracking_number": trackingNumber},
	}
	return s.CreateNotification(ctx, notification)
}

func (s *Service) SendLowStockAlert(ctx context.Context, recipientEmail, productName, sku string, availableQty, threshold int) error {
	subject := fmt.Sprintf("Low Stock Alert: %s (%s)", productName, sku)
	body := fmt.Sprintf("Low stock alert for product %s (SKU: %s).\nCurrent available quantity: %d\nReorder threshold: %d\n\nPlease reorder soon.", productName, sku, availableQty, threshold)

	notification := &Notification{
		Type:       NotificationTypeLowStock,
		Recipient:  recipientEmail,
		Subject:    subject,
		Body:       body,
		Metadata:   map[string]interface{}{"product_name": productName, "sku": sku, "available_qty": availableQty, "threshold": threshold},
	}
	return s.CreateNotification(ctx, notification)
}