package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/util"
)

const (
	ContextUserID      = "userId"
	ContextUsername    = "username"
	ContextDisplayName = "displayName"
	ContextRole        = "role"
)

func Auth(secret string, users repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "请先登录"}, "requestId": c.GetString("requestId")})
			return
		}
		claims, err := util.ParseToken(secret, strings.TrimSpace(parts[1]))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "INVALID_TOKEN", "message": "登录状态已失效"}, "requestId": c.GetString("requestId")})
			return
		}
		user, err := users.Find(c.Request.Context(), claims.UserID)
		if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !user.Active) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "SESSION_REVOKED", "message": "账号已停用或登录状态已撤销"}, "requestId": c.GetString("requestId")})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{"code": "AUTH_LOOKUP_FAILED", "message": "暂时无法校验登录状态"}, "requestId": c.GetString("requestId")})
			return
		}
		c.Set(ContextUserID, user.ID)
		c.Set(ContextUsername, user.Username)
		c.Set(ContextDisplayName, user.DisplayName)
		c.Set(ContextRole, user.Role)
		c.Next()
	}
}
