package billing

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.POST("/billing/checkout", requireUser, handler.Checkout)
	r.GET("/billing/subscription", requireUser, handler.Subscription)
	r.POST("/billing/webhook", handler.Webhook)
	r.GET("/billing/success", handler.SuccessPage)
	r.GET("/billing/cancel", handler.CancelPage)
}
