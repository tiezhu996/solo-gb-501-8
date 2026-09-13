package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"sterile-packaging-release-control/backend/internal/constants"
)

func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, exists := c.Get(ContextRole)
		role, ok := value.(constants.Role)
		if !exists || !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "无法识别当前用户角色"}, "requestId": c.GetString("requestId")})
			return
		}
		if !role.Can(permission) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "当前角色无权执行此操作"}, "requestId": c.GetString("requestId")})
			return
		}
		c.Next()
	}
}

func RequireAnyRole(roles ...constants.Role) gin.HandlerFunc {
	allowed := make(map[constants.Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		role, ok := c.Get(ContextRole)
		if !ok {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if _, exists := allowed[role.(constants.Role)]; !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "角色权限不足"}, "requestId": c.GetString("requestId")})
			return
		}
		c.Next()
	}
}
