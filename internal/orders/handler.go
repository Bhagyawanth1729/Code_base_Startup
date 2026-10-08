package orders

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/forgestock/forgestock/internal/auth"
	"github.com/forgestock/forgestock/internal/inventory"
	"github.com/forgestock/forgestock/internal/platform/http"
)

type Handler struct {
	service   *Service
	validator *validator.Validate
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service:   service,
		validator: validator.New(),
	}
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateOrderRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	order, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, inventory.ErrInsufficientInventory) {
			http.RespondError(c, 400, "insufficient_inventory", err.Error(), "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 201, order)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Order ID is required")
		return
	}

	order, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Order not found")
		return
	}

	userID, _ := auth.GetUserID(c)
	roles, _ := auth.GetUserRoles(c)

	isAdminOrManager := false
	for _, role := range roles {
		if role == "admin" || role == "manager" {
			isAdminOrManager = true
			break
		}
	}

	if !isAdminOrManager && order.CustomerID != userID {
		http.RespondForbidden(c, "Access denied")
		return
	}

	http.Respond(c, 200, order)
}

func (h *Handler) List(c *gin.Context) {
	var query struct {
		Page       int    `form:"page,default=1" validate:"min=1"`
		PageSize   int    `form:"page_size,default=20" validate:"min=1,max=100"`
		CustomerID string `form:"customer_id"`
		Status     string `form:"status" validate:"omitempty,oneof=pending confirmed processing shipped delivered cancelled"`
	}

	if err := http.BindQuery(c, &query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	userID, _ := auth.GetUserID(c)
	roles, _ := auth.GetUserRoles(c)

	isAdminOrManager := false
	for _, role := range roles {
		if role == "admin" || role == "manager" {
			isAdminOrManager = true
			break
		}
	}

	if !isAdminOrManager {
		query.CustomerID = userID
	}

	filter := Filter{
		CustomerID: query.CustomerID,
		Status:     query.Status,
	}

	orders, total, err := h.service.List(c.Request.Context(), filter, query.Page, query.PageSize)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	http.RespondWithMeta(c, 200, orders, &http.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Order ID is required")
		return
	}

	var req UpdateStatusRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			http.RespondNotFound(c, "Order not found")
			return
		}
		if errors.Is(err, ErrInvalidStatusTransition) {
			http.RespondError(c, 400, "invalid_state_transition", err.Error(), "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "Order status updated successfully"})
}

func (h *Handler) Cancel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Order ID is required")
		return
	}

	if err := h.service.Cancel(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			http.RespondNotFound(c, "Order not found")
			return
		}
		if errors.Is(err, ErrInvalidStatusTransition) {
			http.RespondError(c, 400, "invalid_state_transition", err.Error(), "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "Order cancelled successfully"})
}