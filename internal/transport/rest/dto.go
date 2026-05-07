package rest

type ErrorResponse struct {
	Detail string `json:"detail"`
}

type SendMessageRequest struct {
	ChatID string `json:"chat_id" binding:"required"`
	Text   string `json:"text" binding:"required"`
}

type SummaryResponse struct {
    AverageViews          float64 `json:"average_views"`
    AverageReactions      float64 `json:"average_reactions"`
    EngagementRatePercent float64 `json:"engagement_rate_percent"`
    MessagesProcessed     int     `json:"messages_processed"`
    Subscribers           int     `json:"subscribers"`
    ActivityPercentage    float64 `json:"activity_percentage"`
    ContainsKeyword       *bool   `json:"contains_keyword,omitempty"`
}