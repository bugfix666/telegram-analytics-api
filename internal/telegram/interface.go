package telegram

import (
	"context"
	"time"

	"telegram-analytics-api/internal/domain"
)

type ClientInterface interface {
	SendMessage(ctx context.Context, chatID string, text string) (int, error)
	GetChatInfo(ctx context.Context, chatID string) (domain.ChatInfo, error)
	IterateMessages(ctx context.Context, chatID string, since time.Time, limit int) ([]domain.Message, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	WaitReady() <-chan struct{}
}
