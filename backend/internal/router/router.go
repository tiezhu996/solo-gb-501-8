package router

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"sterile-packaging-release-control/backend/internal/config"
	"sterile-packaging-release-control/backend/internal/handler"
	"sterile-packaging-release-control/backend/internal/middleware"
	"sterile-packaging-release-control/backend/internal/repository"
	"sterile-packaging-release-control/backend/internal/service"
	"sterile-packaging-release-control/backend/internal/util"
)

func Build(db *gorm.DB, redisClient *redis.Client, cfg config.Config) (*gin.Engine, error) {
	auditRepo := repository.NewAuditRepository(db)
	userRepo := repository.NewUserRepository(db)
	lineRepo := repository.NewLineRepository(db)
	batchRepo := repository.NewBatchRepository(db)
	inspectionRepo := repository.NewInspectionRepository(db)
	releaseRepo := repository.NewReleaseRepository(db)
	transactor := repository.NewTransactor(db)

	auditService := service.NewAuditService(auditRepo)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.TokenTTL)
	lineService := service.NewLineService(lineRepo, auditService, transactor)
	batchService := service.NewBatchService(batchRepo, lineRepo, auditService, transactor)
	inspectionService := service.NewInspectionService(inspectionRepo, batchRepo, auditService, transactor)
	releaseService := service.NewReleaseService(releaseRepo, batchRepo, inspectionRepo, auditService, transactor)

	if err := authService.Seed(context.Background()); err != nil {
		return nil, err
	}

	authHandler := handler.NewAuthHandler(authService)
	lineHandler := handler.NewLineHandler(lineService)
	batchHandler := handler.NewBatchHandler(batchService)
	inspectionHandler := handler.NewInspectionHandler(inspectionService)
	releaseHandler := handler.NewReleaseHandler(releaseService)
	auditHandler := handler.NewAuditHandler(auditService)

	engine := gin.New()
	engine.Use(middleware.RequestContext())
	engine.Use(middleware.ErrorHandler())
	engine.Use(middleware.NewRateLimiter(redisClient, cfg.RateLimit, cfg.RateWindow).Handler())

	engine.GET("/healthz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := util.Ready(ctx, db); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "database": "unavailable"})
			return
		}
		redisStatus := "ok"
		if err := redisClient.Ping(ctx).Err(); err != nil {
			redisStatus = "fallback"
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "ok", "redis": redisStatus, "service": "sterile-packaging-release-control"})
	})

	api := engine.Group("/api")
	loginLimiter := middleware.NewNamedRateLimiter(redisClient, "login", 10, 5*time.Minute)
	api.POST("/auth/login", loginLimiter.Handler(), authHandler.Login)
	secured := api.Group("")
	secured.Use(middleware.Auth(cfg.JWTSecret, userRepo))
	secured.GET("/auth/me", authHandler.Me)

	secured.GET("/lines", lineHandler.List)
	secured.GET("/lines/:id", lineHandler.Get)
	secured.POST("/lines", middleware.RequirePermission("line:write"), lineHandler.Create)
	secured.PATCH("/lines/:id", middleware.RequirePermission("line:write"), lineHandler.Update)

	secured.GET("/batches", batchHandler.List)
	secured.GET("/overview", batchHandler.Overview)
	secured.GET("/batches/:id", batchHandler.Get)
	secured.POST("/batches", middleware.RequirePermission("batch:write"), batchHandler.Create)
	secured.PATCH("/batches/:id", middleware.RequirePermission("batch:write"), batchHandler.Update)
	secured.POST("/batches/:id/transition", middleware.RequirePermission("batch:write"), batchHandler.Transition)

	secured.GET("/inspections", inspectionHandler.List)
	secured.GET("/inspections/:id", inspectionHandler.Get)
	secured.POST("/inspections", middleware.RequirePermission("inspection:write"), inspectionHandler.Create)
	secured.POST("/inspections/:id/complete", middleware.RequirePermission("inspection:write"), inspectionHandler.Complete)
	secured.POST("/inspections/:id/retest", middleware.RequirePermission("inspection:write"), inspectionHandler.Retest)

	secured.GET("/release-decisions", releaseHandler.List)
	secured.GET("/release-decisions/:id", releaseHandler.Get)
	secured.POST("/release-decisions", middleware.RequirePermission("release:write"), releaseHandler.Decide)
	secured.GET("/audit-logs", middleware.RequirePermission("audit:read"), auditHandler.List)

	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "ROUTE_NOT_FOUND", "message": "接口不存在"}, "requestId": c.GetString("requestId")})
	})
	return engine, nil
}
