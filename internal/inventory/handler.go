package inventory

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/forgestock/forgestock/internal/auth"
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
	var req CreateInventoryRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	inv, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 201, inv)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Inventory ID is required")
		return
	}

	inv, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Inventory not found")
		return
	}

	http.Respond(c, 200, inv)
}

func (h *Handler) GetByProductAndWarehouse(c *gin.Context) {
	productID := c.Query("product_id")
	warehouseID := c.Query("warehouse_id")

	if productID == "" || warehouseID == "" {
		http.RespondValidationError(c, "product_id and warehouse_id are required")
		return
	}

	inv, err := h.service.GetByProductAndWarehouse(c.Request.Context(), productID, warehouseID)
	if err != nil {
		http.RespondNotFound(c, "Inventory not found")
		return
	}

	http.Respond(c, 200, inv)
}

func (h *Handler) ReserveStock(c *gin.Context) {
	var req ReserveStockRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	inv, err := h.service.ReserveStock(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInsufficientInventory) {
			http.RespondError(c, 400, "insufficient_inventory", err.Error(), "")
			return
		}
		if errors.Is(err, ErrOptimisticLock) {
			http.RespondError(c, 409, "optimistic_lock_failed", "Concurrent modification, please retry", "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, inv)
}

func (h *Handler) ReleaseStock(c *gin.Context) {
	var req ReleaseStockRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	inv, err := h.service.ReleaseStock(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrOptimisticLock) {
			http.RespondError(c, 409, "optimistic_lock_failed", "Concurrent modification, please retry", "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, inv)
}

func (h *Handler) AddStock(c *gin.Context) {
	var req AddStockRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	inv, err := h.service.AddStock(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrOptimisticLock) {
			http.RespondError(c, 409, "optimistic_lock_failed", "Concurrent modification, please retry", "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, inv)
}

func (h *Handler) AdjustStock(c *gin.Context) {
	var req AdjustStockRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	inv, err := h.service.AdjustStock(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrNegativeInventory) {
			http.RespondError(c, 400, "negative_inventory", err.Error(), "")
			return
		}
		if errors.Is(err, ErrOptimisticLock) {
			http.RespondError(c, 409, "optimistic_lock_failed", "Concurrent modification, please retry", "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, inv)
}

func (h *Handler) List(c *gin.Context) {
	var query struct {
		Page        int  `form:"page,default=1" validate:"min=1"`
		PageSize    int  `form:"page_size,default=20" validate:"min=1,max=100"`
		ProductID   string `form:"product_id"`
		WarehouseID string `form:"warehouse_id"`
		LowStockOnly bool `form:"low_stock_only"`
	}

	if err := http.BindQuery(c, &query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	filter := Filter{
		ProductID:    query.ProductID,
		WarehouseID:  query.WarehouseID,
		LowStockOnly: query.LowStockOnly,
	}

	inventories, total, err := h.service.List(c.Request.Context(), filter, query.Page, query.PageSize)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	http.RespondWithMeta(c, 200, inventories, &http.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

func (h *Handler) GetTransactions(c *gin.Context) {
	inventoryID := c.Param("id")
	if inventoryID == "" {
		http.RespondValidationError(c, "Inventory ID is required")
		return
	}

	var query struct {
		Page     int `form:"page,default=1" validate:"min=1"`
		PageSize int `form:"page_size,default=20" validate:"min=1,max=100"`
	}

	if err := http.BindQuery(c, &query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	transactions, total, err := h.service.GetTransactions(c.Request.Context(), inventoryID, query.Page, query.PageSize)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	http.RespondWithMeta(c, 200, transactions, &http.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}