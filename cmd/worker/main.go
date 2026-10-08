package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/forgestock/forgestock/internal/auth"
	"github.com/forgestock/forgestock/internal/audit"
	"github.com/forgestock/forgestock/internal/categories"
	"github.com/forgestock/forgestock/internal/inventory"
	"github.com/forgestock/forgestock/internal/notifications"
	"github.com/forgestock/forgestock/internal/orders"
	"github.com/forgestock/forgestock/internal/payments"
	"github.com/forgestock/forgestock/internal/platform/config"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/forgestock/forgestock/internal/platform/redis"
	"github.com/forgestock/forgestock/internal/products"
	"github.com/forgestock/forgestock/internal/users"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.LogLevel, cfg.AppEnv == "development")

	db, err := database.New(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	redisClient, err := redis.New(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()

	userRepo := users.NewRepository(db)
	authService := auth.NewService(db, redisClient, userRepo, auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL), log, cfg.RefreshTokenTTL)

	categoryRepo := categories.NewRepository(db)
	categoryService := categories.NewService(categoryRepo, log)

	productRepo := products.NewRepository(db)
	productService := products.NewService(productRepo, categoryRepo, log)

	inventoryRepo := inventory.NewRepository(db)
	inventoryService := inventory.NewService(inventoryRepo, productRepo, db, log)

	orderRepo := orders.NewRepository(db)
	paymentRepo := payments.NewRepository(db)
	paymentProvider := payments.NewMockProvider()
	paymentService := payments.NewService(paymentRepo, paymentProvider, db, log)
	orderService := orders.NewService(orderRepo, productRepo, inventoryRepo, inventoryService, paymentService, db, log)

	emailSender := notifications.NewConsoleEmailSender()
	notificationService := notifications.NewService(db, redisClient, emailSender, log)

	auditService := audit.NewService(db, log)

	_ = authService
	_ = categoryService
	_ = productService
	_ = inventoryService
	_ = orderService
	_ = paymentService
	_ = notificationService
	_ = auditService

	log.Info().Msg("Starting background worker")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := notificationService.ProcessQueue(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to process notification queue")
				}
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkLowStock(ctx, inventoryService, notificationService, log)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupRevokedTokens(ctx, redisClient, log)
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down worker")
	cancel()
	time.Sleep(2 * time.Second)
	log.Info().Msg("Worker stopped")
}

func checkLowStock(ctx context.Context, inventoryService *inventory.Service, notificationService *notifications.Service, log *logger.Logger) {
	inv, total, err := inventoryService.List(ctx, inventory.Filter{LowStockOnly: true}, 1, 100)
	if err != nil {
		log.Error().Err(err).Msg("Failed to check low stock")
		return
	}

	for _, i := range inv {
		product, err := inventoryService.GetByID(ctx, i.ID)
		if err != nil {
			continue
		}
		// In a real app, you'd get product name from product service
		notificationService.SendLowStockAlert(ctx, "admin@forgestock.com", "Product "+i.ProductID, i.ProductID, i.AvailableQty, i.ReorderThreshold)
	}

	if total > 0 {
		log.Info().Int("low_stock_count", total).Msg("Low stock alert check completed")
	}
}

func cleanupRevokedTokens(ctx context.Context, redisClient *redis.Client, log *logger.Logger) {
	// Cleanup old revoked tokens (handled by TTL in Redis)
	log.Debug().Msg("Revoked token cleanup check completed")
}