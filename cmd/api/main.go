package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/forgestock/forgestock/internal/auth"
	"github.com/forgestock/forgestock/internal/audit"
	"github.com/forgestock/forgestock/internal/categories"
	"github.com/forgestock/forgestock/internal/health"
	"github.com/forgestock/forgestock/internal/inventory"
	"github.com/forgestock/forgestock/internal/orders"
	"github.com/forgestock/forgestock/internal/payments"
	"github.com/forgestock/forgestock/internal/platform/config"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/http/middleware"
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

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.LoggerMiddleware(log))
	router.Use(middleware.RecoveryMiddleware(log))
	router.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))
	router.Use(middleware.SecurityHeadersMiddleware())

	userRepo := users.NewRepository(db)
	userService := users.NewService(userRepo, log)
	userHandler := users.NewHandler(userService)

	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authService := auth.NewService(db, redisClient, userRepo, jwtManager, log, cfg.RefreshTokenTTL)
	authHandler := auth.NewHandler(authService)

	categoryRepo := categories.NewRepository(db)
	categoryService := categories.NewService(categoryRepo, log)
	categoryHandler := categories.NewHandler(categoryService)

	productRepo := products.NewRepository(db)
	productService := products.NewService(productRepo, categoryRepo, log)
	productHandler := products.NewHandler(productService)

	inventoryRepo := inventory.NewRepository(db)
	inventoryService := inventory.NewService(inventoryRepo, productRepo, db, log)
	inventoryHandler := inventory.NewHandler(inventoryService)

	orderRepo := orders.NewRepository(db)
	paymentRepo := payments.NewRepository(db)
	paymentProvider := payments.NewMockProvider()
	paymentService := payments.NewService(paymentRepo, paymentProvider, db, log)
	orderService := orders.NewService(orderRepo, productRepo, inventoryRepo, inventoryService, paymentService, db, log)
	orderHandler := orders.NewHandler(orderService)

	paymentHandler := payments.NewHandler(paymentService)

	auditService := audit.NewService(db, log)
	_ = auditService

	healthHandler := health.NewHandler(db, redisClient, log)

	api := router.Group("/api/v1")
	{
		healthGroup := api.Group("/health")
		{
			healthGroup.GET("", healthHandler.Health)
			healthGroup.GET("/ready", healthHandler.Ready)
		}

		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/refresh", authHandler.Refresh)
			authGroup.POST("/logout", authMiddleware(jwtManager, log), authHandler.Logout)
			authGroup.POST("/change-password", authMiddleware(jwtManager, log), authHandler.ChangePassword)
		}

		userGroup := api.Group("/users")
		userGroup.Use(authMiddleware(jwtManager, log))
		{
			userGroup.GET("/me", userHandler.GetProfile)
			userGroup.PATCH("/me", userHandler.UpdateProfile)
		}

		adminUserGroup := api.Group("/users")
		adminUserGroup.Use(authMiddleware(jwtManager, log))
		adminUserGroup.Use(auth.RequireRoles("admin", "manager"))
		{
			adminUserGroup.GET("", userHandler.ListUsers)
			adminUserGroup.GET("/:id", userHandler.GetUser)
			adminUserGroup.PATCH("/:id/role", userHandler.UpdateUserRole)
			adminUserGroup.POST("/:id/deactivate", userHandler.DeactivateUser)
			adminUserGroup.POST("/:id/activate", userHandler.ActivateUser)
		}

		categoryGroup := api.Group("/categories")
		categoryGroup.Use(authMiddleware(jwtManager, log))
		{
			categoryGroup.GET("", categoryHandler.List)
			categoryGroup.GET("/:id", categoryHandler.GetByID)
		}

		adminCategoryGroup := api.Group("/categories")
		adminCategoryGroup.Use(authMiddleware(jwtManager, log))
		adminCategoryGroup.Use(auth.RequireRoles("admin", "manager"))
		{
			adminCategoryGroup.POST("", categoryHandler.Create)
			adminCategoryGroup.PATCH("/:id", categoryHandler.Update)
			adminCategoryGroup.DELETE("/:id", categoryHandler.Delete)
		}

		productGroup := api.Group("/products")
		productGroup.Use(authMiddleware(jwtManager, log))
		{
			productGroup.GET("", productHandler.List)
			productGroup.GET("/:id", productHandler.GetByID)
		}

		adminProductGroup := api.Group("/products")
		adminProductGroup.Use(authMiddleware(jwtManager, log))
		adminProductGroup.Use(auth.RequireRoles("admin", "manager"))
		{
			adminProductGroup.POST("", productHandler.Create)
			adminProductGroup.PATCH("/:id", productHandler.Update)
			adminProductGroup.DELETE("/:id", productHandler.Delete)
		}

		inventoryGroup := api.Group("/inventory")
		inventoryGroup.Use(authMiddleware(jwtManager, log))
		inventoryGroup.Use(auth.RequireRoles("admin", "manager", "warehouse_operator"))
		{
			inventoryGroup.GET("", inventoryHandler.List)
			inventoryGroup.GET("/:id", inventoryHandler.GetByID)
			inventoryGroup.GET("/by-product", inventoryHandler.GetByProductAndWarehouse)
			inventoryGroup.GET("/:id/transactions", inventoryHandler.GetTransactions)
			inventoryGroup.POST("/reserve", inventoryHandler.ReserveStock)
			inventoryGroup.POST("/release", inventoryHandler.ReleaseStock)
			inventoryGroup.POST("/add", inventoryHandler.AddStock)
			inventoryGroup.POST("/adjust", inventoryHandler.AdjustStock)
		}

		orderGroup := api.Group("/orders")
		orderGroup.Use(authMiddleware(jwtManager, log))
		{
			orderGroup.POST("", orderHandler.Create)
			orderGroup.GET("", orderHandler.List)
			orderGroup.GET("/:id", orderHandler.GetByID)
			orderGroup.POST("/:id/cancel", orderHandler.Cancel)
		}

		adminOrderGroup := api.Group("/orders")
		adminOrderGroup.Use(authMiddleware(jwtManager, log))
		adminOrderGroup.Use(auth.RequireRoles("admin", "manager"))
		{
			adminOrderGroup.PATCH("/:id/status", orderHandler.UpdateStatus)
		}

		paymentGroup := api.Group("/payments")
		paymentGroup.Use(authMiddleware(jwtManager, log))
		{
			paymentGroup.POST("/:id/authorize", paymentHandler.Authorize)
			paymentGroup.POST("/:id/capture", paymentHandler.Capture)
			paymentGroup.POST("/:id/refund", paymentHandler.Refund)
			paymentGroup.POST("/:id/void", paymentHandler.Void)
		}
	}

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.HTTPPort).Msg("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info().Msg("Shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}

func authMiddleware(jwtManager *auth.JWTManager, log *logger.Logger) gin.HandlerFunc {
	return auth.AuthMiddleware(jwtManager, log)
}