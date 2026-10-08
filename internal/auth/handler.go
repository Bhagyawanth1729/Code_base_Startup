package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/forgestock/forgestock/internal/platform/http"
	"github.com/forgestock/forgestock/internal/users"
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

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	tokens, err := h.service.Register(c.Request.Context(), req)
	if err != nil {
		if err == ErrInvalidCredentials || err == users.ErrEmailExists {
			http.RespondConflict(c, err.Error())
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 201, tokens)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	tokens, err := h.service.Login(c.Request.Context(), req)
	if err != nil {
		if err == ErrInvalidCredentials {
			http.RespondUnauthorized(c, "Invalid email or password")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, tokens)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	tokens, err := h.service.Refresh(c.Request.Context(), req)
	if err != nil {
		if err == ErrTokenRevoked || err.Error() == "refresh token expired" || err.Error() == "invalid refresh token" {
			http.RespondUnauthorized(c, err.Error())
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, tokens)
}

func (h *Handler) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		http.RespondUnauthorized(c, "Authorization header required")
		return
	}

	parts := splitAuthHeader(authHeader)
	if len(parts) != 2 {
		http.RespondUnauthorized(c, "Invalid authorization header")
		return
	}

	if err := h.service.Logout(c.Request.Context(), parts[1]); err != nil {
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "Logged out successfully"})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		http.RespondUnauthorized(c, "User ID not found")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" validate:"required"`
		NewPassword     string `json:"new_password" validate:"required,min=8,max=72"`
	}

	if err := http.BindJSON(c, &req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.validator.Struct(req); err != nil {
		http.RespondValidationError(c, err.Error())
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if err == ErrInvalidCredentials {
			http.RespondUnauthorized(c, "Current password is incorrect")
			return
		}
		http.RespondInternalError(c, nil, err)
		return
	}

	http.Respond(c, 200, gin.H{"message": "Password changed successfully"})
}

func splitAuthHeader(header string) []string {
	for i := 0; i < len(header); i++ {
		if header[i] == ' ' {
			return []string{header[:i], header[i+1:]}
		}
	}
	return []string{header}
}