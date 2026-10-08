package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/forgestock/forgestock/internal/platform/database"
	"github.com/forgestock/forgestock/internal/platform/logger"
	"github.com/forgestock/forgestock/internal/platform/redis"
)

type Handler struct {
	db    *database.DB
	redis *redis.Client
	log   *logger.Logger
}

func NewHandler(db *database.DB, redisClient *redis.Client, log *logger.Logger) *Handler {
	return &Handler{db: db, redis: redisClient, log: log}
}

func (h *Handler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		h.log.Error().Err(err).Msg("Health check: database ping failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"checks": gin.H{
				"database": "failed",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"checks": gin.H{
			"database": "ok",
		},
	})
}

func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := gin.H{}
	allReady := true

	if err := h.db.Ping(ctx); err != nil {
		checks["database"] = "not ready"
		allReady = false
	} else {
		checks["database"] = "ready"
	}

	if err := h.redis.HealthCheck(ctx); err != nil {
		checks["redis"] = "not ready"
		allReady = false
	} else {
		checks["redis"] = "ready"
	}

	status := http.StatusOK
	if !allReady {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status": "ready",
		"checks": checks,
	})
}

func (h *Handler) Metrics(c *gin.Context) {
	// Metrics are exposed via promhttp.Handler() in main.go
	c.Status(http.StatusNotFound)
}