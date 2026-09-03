package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Data any `json:"data"`
}

type PaginatedResponse struct {
	Data  any   `json:"data"`
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int   `json:"total"`
}

type ErrorResponse struct {
	Code      string        `json:"code"`
	Message   string        `json:"message"`
	RequestID string        `json:"request_id"`
	Details   []ErrorDetail `json:"details,omitempty"`
}

type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func RespondSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, SuccessResponse{Data: data})
}

func RespondCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, SuccessResponse{Data: data})
}

func RespondNoContent(c *gin.Context) {
	c.Writer.WriteHeader(http.StatusNoContent)
}

func RespondPaginated(c *gin.Context, data any, page, limit, total int) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Data:  data,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func RespondError(c *gin.Context, status int, code, message string, details ...ErrorDetail) {
	requestID, _ := c.Get("requestID")
	requestIDStr, _ := requestID.(string)

	resp := ErrorResponse{
		Code:      code,
		Message:   message,
		RequestID: requestIDStr,
		Details:   details,
	}
	c.JSON(status, resp)
}

func RespondValidationError(c *gin.Context, details []ErrorDetail) {
	RespondError(c, http.StatusUnprocessableEntity, "validation_error", "Validation failed", details...)
}

func RespondNotFound(c *gin.Context, message string) {
	RespondError(c, http.StatusNotFound, "not_found", message)
}

func RespondConflict(c *gin.Context, message string) {
	RespondError(c, http.StatusConflict, "conflict", message)
}

func RespondUnauthorized(c *gin.Context, message string) {
	RespondError(c, http.StatusUnauthorized, "unauthorized", message)
}

func RespondRateLimited(c *gin.Context) {
	RespondError(c, http.StatusTooManyRequests, "rate_limited", "Rate limit exceeded")
}

func RespondInternalError(c *gin.Context) {
	RespondError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
}
