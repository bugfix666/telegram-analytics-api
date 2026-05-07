package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bugfix666/telegram-analytics-api/internal/domain"
)

type mockClient struct {
	chatInfo  domain.ChatInfo
	messages  []domain.Message
	err       error
	sendMsgID int
	sendErr   error
}

func (m *mockClient) SendMessage(ctx context.Context, chatID, text string) (int, error) {
	return m.sendMsgID, m.sendErr
}
func (m *mockClient) GetChatInfo(ctx context.Context, chatID string) (domain.ChatInfo, error) {
	return m.chatInfo, m.err
}
func (m *mockClient) IterateMessages(ctx context.Context, chatID string, since time.Time, limit int) ([]domain.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	var filtered []domain.Message
	for _, msg := range m.messages {
		if since.IsZero() || msg.Date.After(since) || msg.Date.Equal(since) {
			filtered = append(filtered, msg)
		}
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
}
func (m *mockClient) Start(ctx context.Context) error { return nil }
func (m *mockClient) Stop(ctx context.Context) error  { return nil }
func (m *mockClient) WaitReady() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func TestGetSummary(t *testing.T) {
	now := time.Now()
	messages := []domain.Message{
		{ID: 1, Date: now.Add(-5 * 24 * time.Hour), Views: 150, Reactions: map[string]int{"👍": 10}, Text: "No keyword here"},
		{ID: 2, Date: now.Add(-15 * 24 * time.Hour), Views: 200, Reactions: map[string]int{"❤️": 5}, Text: "This contains COLLABA25"},
		{ID: 3, Date: now.Add(-40 * 24 * time.Hour), Views: 100, Reactions: nil, Text: "Another without"},
	}
	chatInfo := domain.ChatInfo{
		ID:          "@test",
		Subscribers: 1000,
		Description: "Test channel COLLABA25",
	}
	mock := &mockClient{
		chatInfo: chatInfo,
		messages: messages,
	}
	svc := NewAnalyticsService(mock)

	summary, err := svc.GetSummary(context.Background(), "@test", 10, "COLLABA25")
	assert.NoError(t, err)
	assert.Equal(t, 1000, summary.Subscribers)
	assert.NotNil(t, summary.ContainsKeyword)
	assert.True(t, *summary.ContainsKeyword)
	assert.Equal(t, 1, summary.MessagesProcessed)
	assert.Equal(t, 200.0, summary.AverageViews)
	assert.Equal(t, 5.0, summary.AverageReactions)

	summary2, err := svc.GetSummary(context.Background(), "@test", 10, "")
	assert.NoError(t, err)
	assert.Nil(t, summary2.ContainsKeyword)
	assert.Equal(t, 3, summary2.MessagesProcessed)
	assert.Equal(t, 150.0, summary2.AverageViews)   // (150+200+100)/3 = 150
	assert.Equal(t, 7.5, summary2.AverageReactions) // (10+5)/2 = 7.5
}
