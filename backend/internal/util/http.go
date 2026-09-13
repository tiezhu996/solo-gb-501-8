package util

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string { return e.Message }

func BadRequest(message string) error {
	return &APIError{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: message}
}

func Forbidden(message string) error {
	return &APIError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: message}
}

func Conflict(message string) error {
	return &APIError{Status: http.StatusConflict, Code: "CONFLICT", Message: message}
}

func NotFound(message string) error {
	return &APIError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: message}
}

func Respond(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data, "requestId": c.GetString("requestId")})
}

func RespondError(c *gin.Context, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		c.JSON(apiErr.Status, gin.H{"error": gin.H{"code": apiErr.Code, "message": apiErr.Message}, "requestId": c.GetString("requestId")})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "记录不存在"}, "requestId": c.GetString("requestId")})
		return
	}
	_ = c.Error(err)
}

func BindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		RespondError(c, BadRequest(err.Error()))
		return false
	}
	return true
}

func ParseID(c *gin.Context) (uint, bool) {
	var target struct {
		ID uint `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&target); err != nil || target.ID == 0 {
		RespondError(c, BadRequest("无效的资源 ID"))
		return 0, false
	}
	return target.ID, true
}
