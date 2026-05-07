package telegram

type AnalyticsResult struct {
	SourceType        string  `json:"source_type"`
	ID                string  `json:"id"`
	AverageViews      float64 `json:"average_views"`
	AverageReactions  float64 `json:"average_reactions"`
	EngagementRate    float64 `json:"engagement_rate_percent"`
	MessagesProcessed int     `json:"messages_processed"`
	Subscribers       int     `json:"subscribers"`
	ActivityPercent   float64 `json:"activity_percentage"`
	ContainsKeyword   *bool   `json:"contains_keyword,omitempty"`
}
