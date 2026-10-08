package products

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
	var req CreateProductRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	product, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrSKUExists) {
			http.RespondConflict(c, err.Error())
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 201, product)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Product ID is required")
		return
	}

	product, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Product not found")
		return
	}

	http.Respond(c, 200, product)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Product ID is required")
		return
	}

	var req UpdateProductRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	product, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrSKUExists) {
			http.RespondConflict(c, err.Error())
			return
		}
		if errors.Is(err, ErrProductNotFound) {
			http.RespondNotFound(c, "Product not found")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, product)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Product ID is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			http.RespondNotFound(c, "Product not found")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "Product deleted successfully"})
}

func (h *Handler) List(c *gin.Context) {
	var query struct {
		Page       int     `form:"page,default=1" validate:"min=1"`
		PageSize   int     `form:"page_size,default=20" validate:"min=1,max=100"`
		CategoryID string  `form:"category_id"`
		Active     *bool   `form:"active"`
		Search     string  `form:"search"`
		SortBy     string  `form:"sort_by"`
		SortOrder  string  `form:"sort_order,default=DESC" validate:"oneof=ASC DESC"`
	}

	if err := http.BindQuery(c, &query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(query); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	filter := ProductFilter{
		CategoryID: query.CategoryID,
		Active:     query.Active,
		Search:     query.Search,
		SortBy:     query.SortBy,
		SortOrder:  query.SortOrder,
	}

	products, total, err := h.service.List(c.Request.Context(), filter, query.Page, query.PageSize)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	http.RespondWithMeta(c, 200, products, &http.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}