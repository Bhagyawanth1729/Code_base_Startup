package payments

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

func (h *Handler) Authorize(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Payment ID is required")
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

	payment, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Payment not found")
		return
	}

	if !isAdminOrManager {
		// In a real app, check if user owns the order
		http.RespondForbidden(c, "Access denied")
		return
	}

	payment, err = h.service.AuthorizePayment(c.Request.Context(), id)
	if err != nil {
		if err == ErrPaymentNotPending {
			http.RespondError(c, 400, "invalid_state", err.Error(), "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, payment)
}

func (h *Handler) Capture(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Payment ID is required")
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

	payment, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Payment not found")
		return
	}

	if !isAdminOrManager {
		http.RespondForbidden(c, "Access denied")
		return
	}

	payment, err = h.service.CapturePayment(c.Request.Context(), id)
	if err != nil {
		if err == ErrPaymentNotAuthorized {
			http.RespondError(c, 400, "invalid_state", err.Error(), "")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, payment)
}

func (h *Handler) Refund(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Payment ID is required")
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

	payment, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Payment not found")
		return
	}

	if !isAdminOrManager {
		http.RespondForbidden(c, "Access denied")
		return
	}

	payment, err = h.service.RefundPayment(c.Request.Context(), id)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, payment)
}

func (h *Handler) Void(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		http.RespondValidationError(c, "Payment ID is required")
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

	payment, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		http.RespondNotFound(c, "Payment not found")
		return
	}

	if !isAdminOrManager {
		http.RespondForbidden(c, "Access denied")
		return
	}

	payment, err = h.service.VoidPayment(c.Request.Context(), id)
	if err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, payment)
}