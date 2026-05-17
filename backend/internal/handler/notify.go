package handler

import (
	"net/http"

	"confirmation-review-service/internal/repository"

	"github.com/gin-gonic/gin"
)

type PushSubscriptionRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
	Keys     struct {
		P256DH string `json:"p256dh" binding:"required"`
		Auth   string `json:"auth" binding:"required"`
	} `json:"keys" binding:"required"`
}

func SubscribePush(c *gin.Context) {
	var req PushSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email := c.GetString("user_email")

	if err := repository.SavePushSubscription(email, req.Endpoint, req.Keys.P256DH, req.Keys.Auth); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true})
}

func VAPIDPublicKeyHandler(publicKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"publicKey": publicKey})
	}
}
