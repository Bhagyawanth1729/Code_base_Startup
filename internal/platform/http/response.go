package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/forgestock/forgestock/internal/platform/logger"
)

type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
	Meta      *Meta       `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type Meta struct {
	Page       int `json:"page,omitempty"`
	PageSize   int `json:"page_size,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

func Respond(c *gin.Context, status int, data interface{}) {
	requestID := c.GetString("request_id")
	c.JSON(status, APIResponse{
		Success:   true,
		Data:      data,
		RequestID: requestID,
	})
}

func RespondWithMeta(c *gin.Context, status int, data interface{}, meta *Meta) {
	requestID := c.GetString("request_id")
	c.JSON(status, APIResponse{
		Success:   true,
		Data:      data,
		RequestID: requestID,
		Meta:      meta,
	})
}

func RespondError(c *gin.Context, status int, code, message, details string) {
	requestID := c.GetString("request_id")
	c.JSON(status, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		RequestID: requestID,
	})
}

func RespondValidationError(c *gin.Context, details string) {
	RespondError(c, http.StatusBadRequest, "validation_error", "Validation failed", details)
}

func RespondUnauthorized(c *gin.Context, message string) {
	RespondError(c, http.StatusUnauthorized, "unauthorized", message, "")
}

func RespondForbidden(c *gin.Context, message string) {
	RespondError(c, http.StatusForbidden, "forbidden", message, "")
}

func RespondNotFound(c *gin.Context, message string) {
	RespondError(c, http.StatusNotFound, "not_found", message, "")
}

func RespondConflict(c *gin.Context, message string) {
	RespondError(c, http.StatusConflict, "conflict", message, "")
}

func RespondInternalError(c *gin.Context, log *logger.Logger, err error) {
	requestID := c.GetString("request_id")
	log.Error().
		Str("request_id", requestID).
		Err(err).
		Msg("Internal server error")
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "internal_error",
			Message: "An unexpected error occurred",
		},
		RequestID: requestID,
	})
}

func BindJSON(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return err
	}
	return nil
}

func BindQuery(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindQuery(obj); err != nil {
		return err
	}
	return nil
}

func BindUri(c *gin.Context, obj interface{}) error {
	if err := c.ShouldBindUri(obj); err != nil {
		return err
	}
	return nil
}