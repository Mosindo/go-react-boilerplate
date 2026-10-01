package users

import "github.com/gin-gonic/gin"

// Account self-service. There is deliberately no endpoint that lists users:
// other members are only reachable through discovery, matches and chat.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, sensitiveLimit gin.HandlerFunc) {
	r.DELETE("/me", requireUser, sensitiveLimit, handler.DeleteAccount)
}
