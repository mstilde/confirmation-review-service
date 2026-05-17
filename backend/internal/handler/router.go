package handler

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"confirmation-review-service/internal/auth"
	"confirmation-review-service/internal/config"
	"confirmation-review-service/internal/service"
)

func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "x-bridge-key"},
		AllowCredentials: true,
	}))

	caseSvc := service.NewCaseService(cfg.N8NPendingActionWebhookURL)
	caseHandler := NewCaseHandler(caseSvc)
	authHandler := NewAuthHandler(cfg.JWTSecret)

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/setup", authHandler.Setup)
		authGroup.GET("/me", authHandler.Me)
	}

	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		c.Set("user_email", "anonymous")
		c.Next()
	})
	{
		api.GET("/cases/pending", caseHandler.ListPending)
		api.GET("/cases/:id", caseHandler.GetByID)
		api.POST("/cases/:id/approve", caseHandler.Approve)
		api.POST("/cases/:id/skip", caseHandler.Skip)
		api.POST("/cases/:id/cancel", caseHandler.Cancel)

		api.POST("/push/subscribe", SubscribePush)
		api.GET("/push/vapid-public-key", VAPIDPublicKeyHandler(cfg.VAPIDPublicKey))
	}

	bridge := r.Group("/api")
	bridge.Use(auth.BridgeKeyAuth(cfg.BridgeKey))
	{
		bridge.POST("/cases", caseHandler.Create)
		bridge.POST("/cases/:id/refresh-chat", caseHandler.RefreshChat)
		bridge.POST("/notify", caseHandler.Notify)
		bridge.GET("/cases/count", caseHandler.CountPending)
		bridge.POST("/cases/expire", caseHandler.ExpireOld)
	}

	return r
}
