package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

func RequestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if !requestIDPattern.MatchString(requestID) {
			requestID = uuid.NewString()
		}
		c.Set("requestId", requestID)
		c.Header("X-Request-ID", requestID)
		started := time.Now()
		c.Next()
		slog.Info("request", "requestId", requestID, "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "durationMs", time.Since(started).Milliseconds())
	}
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic recovered", "error", fmt.Sprint(recovered), "requestId", c.GetString("requestId"), "stack", string(debug.Stack()))
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "服务暂时不可用"}, "requestId": c.GetString("requestId")})
			}
		}()
		c.Next()
		if len(c.Errors) > 0 && !c.Writer.Written() {
			err := c.Errors.Last().Err
			slog.Error("request failed", "error", err, "requestId", c.GetString("requestId"))
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "服务器处理请求失败"}, "requestId": c.GetString("requestId")})
		}
	}
}
