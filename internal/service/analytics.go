package service

import (
	"context"
	"strings"
	"time"

	"telegram-analytics-api/internal/domain"
	"telegram-analytics-api/internal/telegram"
)

type AnalyticsService struct {
	client telegram.ClientInterface
}

func NewAnalyticsService(client telegram.ClientInterface) *AnalyticsService {
	return &AnalyticsService{client: client}
}

func (s *AnalyticsService) SendMessage(ctx context.Context, chatID, text string) (int, error) {
	return s.client.SendMessage(ctx, chatID, text)
}

func (s *AnalyticsService) GetSummary(ctx context.Context, chatID string, limit int, keyword string) (domain.AnalyticsSummary, error) {
	var summary domain.AnalyticsSummary

	chatInfo, err := s.client.GetChatInfo(ctx, chatID)
	if err != nil {
		return summary, err
	}
	summary.Subscribers = chatInfo.Subscribers

	if keyword != "" {
		ok := strings.Contains(chatInfo.Description, keyword)
		summary.ContainsKeyword = &ok
	}

	messages, err := s.client.IterateMessages(ctx, chatID, time.Time{}, limit)
	if err != nil {
		return summary, err
	}

	filteredMessages := messages
	if keyword != "" {
		filtered := make([]domain.Message, 0)
		for _, msg := range messages {
			if strings.Contains(msg.Text, keyword) {
				filtered = append(filtered, msg)
			}
		}
		filteredMessages = filtered
	}

	summary.MessagesProcessed = len(filteredMessages)

	var totalViews, totalReactions int64
	var reactionCount int
	for _, msg := range filteredMessages {
		if msg.Views > 0 {
			totalViews += int64(msg.Views)
		}
		if len(msg.Reactions) > 0 {
			var sum int64
			for _, cnt := range msg.Reactions {
				sum += int64(cnt)
			}
			totalReactions += sum
			reactionCount++
		}
	}

	if len(filteredMessages) > 0 {
		summary.AverageViews = float64(totalViews) / float64(len(filteredMessages))
		if reactionCount > 0 {
			summary.AverageReactions = float64(totalReactions) / float64(reactionCount)
		}
	}

	if totalViews > 0 {
		summary.EngagementRate = (float64(totalReactions) / float64(totalViews)) * 100
	}

	if summary.Subscribers > 0 {
		summary.ActivityPercent = (summary.AverageViews / float64(summary.Subscribers)) * 100
	}

	return summary, nil
}
