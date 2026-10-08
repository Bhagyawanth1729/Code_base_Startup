package users

import (
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

func (h *Handler) GetProfile(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		http.RespondUnauthorized(c, "User ID not found")
		return
	}

	user, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		http.RespondNotFound(c, "User not found")
		return
	}

	http.Respond(c, 200, user)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		http.RespondUnauthorized(c, "User ID not found")
		return
	}

	var req UpdateProfileRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, user)
}

func (h *Handler) ListUsers(c *gin.Context) {
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

	users, total, err := h.service.ListUsers(c.Request.Context(), query.Page, query.PageSize)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	totalPages := (total + query.PageSize - 1) / query.PageSize
	http.RespondWithMeta(c, 200, users, &http.Meta{
		Page:       query.Page,
		PageSize:   query.PageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

func (h *Handler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		http.RespondValidationError(c, "User ID is required")
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		http.RespondNotFound(c, "User not found")
		return
	}

	http.Respond(c, 200, user)
}

func (h *Handler) UpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		http.RespondValidationError(c, "User ID is required")
		return
	}

	var req struct {
		Role string `json:"role" validate:"required,oneof=admin manager warehouse_operator customer"`
	}

	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.service.UpdateUserRole(c.Request.Context(), userID, req.Role); err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "Role updated successfully"})
}

func (h *Handler) DeactivateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		http.RespondValidationError(c, "User ID is required")
		return
	}

	if err := h.service.DeactivateUser(c.Request.Context(), userID); err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "User deactivated successfully"})
}

func (h *Handler) ActivateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		http.RespondValidationError(c, "User ID is required")
		return
	}

	if err := h.service.ActivateUser(c.Request.Context(), userID); err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "User activated successfully"})
}