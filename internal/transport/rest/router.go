package rest

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRouter(h *Handler, log *zap.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(LoggerMiddleware(log))

	r.POST("/send_message/", h.SendMessage)
	r.GET("/get/", h.GetSummary)

	return r
}