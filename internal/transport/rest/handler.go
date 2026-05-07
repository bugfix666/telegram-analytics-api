package rest

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bugfix666/telegram-analytics-api/internal/service"
)

type Handler struct {
	svc *service.AnalyticsService
}

func NewHandler(svc *service.AnalyticsService) *Handler {
	return &Handler{svc: svc}
}

func roundTwo(v float64) float64 {
	return math.Round(v*100) / 100
}

func (h *Handler) SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Detail: err.Error()})
		return
	}
	id, err := h.svc.SendMessage(c.Request.Context(), req.ChatID, req.Text)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Detail: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message_id": id, "status": "sent"})
}

func (h *Handler) GetSummary(c *gin.Context) {
	chatID := c.Query("group_id")
	if chatID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Detail: "missing group_id"})
		return
	}

	limit := 1000
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	keyword := c.Query("keyword")

	summary, err := h.svc.GetSummary(c.Request.Context(), chatID, limit, keyword)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Detail: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SummaryResponse{
		AverageViews:          roundTwo(summary.AverageViews),
		AverageReactions:      roundTwo(summary.AverageReactions),
		EngagementRatePercent: roundTwo(summary.EngagementRate),
		MessagesProcessed:     summary.MessagesProcessed,
		Subscribers:           summary.Subscribers,
		ActivityPercentage:    roundTwo(summary.ActivityPercent),
		ContainsKeyword:       summary.ContainsKeyword,
	})
}
