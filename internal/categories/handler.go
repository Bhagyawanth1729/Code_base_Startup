package categories

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
	var req CreateCategoryRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	category, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrCategoryNameExists) {
			http.RespondConflict(c, err.Error())
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 201, category)
}

func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Category ID is required")
		return
	}

	category, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Category not found")
		return
	}

	http.Respond(c, 200, category)
}

func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Category ID is required")
		return
	}

	var req UpdateCategoryRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	category, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrCategoryNameExists) {
			http.RespondConflict(c, err.Error())
			return
		}
		if errors.Is(err, ErrCategoryNotFound) {
			http.RespondNotFound(c, "Category not found")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, category)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Category ID is required")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrCategoryInUse) {
			http.RespondConflict(c, err.Error())
			return
		}
		if errors.Is(err, ErrCategoryNotFound) {
			http.RespondNotFound(c, "Category not found")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "Category deleted successfully"})
}

func (h *Handler) List(c *gin.Context) {
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

	categories, total, err := h.service.List(c.Request.Context(), query.Page, query.PageSize)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	http.RespondWithMeta(c, 200, categories, &http.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}