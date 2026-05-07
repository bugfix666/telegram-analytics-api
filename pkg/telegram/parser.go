package telegram

import (
	"context"

	"go.uber.org/zap"

	"telegram-analytics-api/internal/service"
	"telegram-analytics-api/internal/telegram"
)

type Parser struct {
	client  *telegram.Client
	service *service.AnalyticsService
	log     *zap.Logger
}

func NewParser(opts Options, log *zap.Logger) *Parser {
	client := telegram.NewClient(opts.APIID, opts.APIHash, opts.SessionFile, opts.BotToken, opts.Proxy, log)
	svc := service.NewAnalyticsService(client)
	return &Parser{
		client:  client,
		service: svc,
		log:     log,
	}
}

func (p *Parser) Name() string {
	return "telegram"
}

func (p *Parser) Init(ctx context.Context) error {
	go func() {
		if err := p.client.Start(ctx); err != nil && err != context.Canceled {
			p.log.Error("Telegram client failed", zap.Error(err))
		}
	}()
	return nil
}

func (p *Parser) WaitReady() <-chan struct{} {
	return p.client.WaitReady()
}

func (p *Parser) Analyze(ctx context.Context, id string, limit int, keyword string) (AnalyticsResult, error) {
	summary, err := p.service.GetSummary(ctx, id, limit, keyword)
	if err != nil {
		return AnalyticsResult{}, err
	}
	return AnalyticsResult{
		SourceType:        p.Name(),
		ID:                id,
		AverageViews:      summary.AverageViews,
		AverageReactions:  summary.AverageReactions,
		EngagementRate:    summary.EngagementRate,
		MessagesProcessed: summary.MessagesLast30d,
		Subscribers:       summary.Subscribers,
		ActivityPercent:   summary.ActivityPercent,
		ContainsKeyword:   summary.ContainsKeyword,
	}, nil
}

func (p *Parser) SendMessage(ctx context.Context, id, text string) (int, error) {
	return p.service.SendMessage(ctx, id, text)
}
