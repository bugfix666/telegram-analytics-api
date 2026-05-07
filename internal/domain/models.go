package domain

import "time"

type Message struct {
    ID        int
    Date      time.Time
    Views     int
    Forwards  int
    Reactions map[string]int
    Text      string
}

type ChatInfo struct {
    ID          string
    Title       string
    Subscribers int
    Description string
}

type AnalyticsSummary struct {
    AverageViews      float64
    AverageReactions  float64
    EngagementRate    float64
    MessagesProcessed   int
    Subscribers       int
    ActivityPercent   float64
    ContainsKeyword   *bool
}