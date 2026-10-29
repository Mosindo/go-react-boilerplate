package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, sendLimit gin.HandlerFunc) {
	if sendLimit == nil {
		sendLimit = func(c *gin.Context) { c.Next() }
	}
	r.GET("/conversations", requireUser, handler.ListConversations)
	r.GET("/conversations/:id/messages", requireUser, handler.ListMessages)
	r.POST("/conversations/:id/messages", requireUser, sendLimit, handler.Send)
	r.POST("/conversations/:id/read", requireUser, handler.MarkRead)
	r.DELETE("/conversations/:id", requireUser, handler.Hide)
}
