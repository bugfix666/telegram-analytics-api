package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/bugfix666/telegram-analytics-api/internal/config"
	"github.com/bugfix666/telegram-analytics-api/internal/logger"
	"github.com/bugfix666/telegram-analytics-api/internal/service"
	"github.com/bugfix666/telegram-analytics-api/internal/telegram"
	"github.com/bugfix666/telegram-analytics-api/internal/transport/rest"
)

func main() {
	cfg := config.Load()
	log := logger.New()
	defer log.Sync()
	tgClient := telegram.NewClient(cfg.APIID, cfg.APIHash, cfg.SessionFile, cfg.BotToken, cfg.Proxy, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := tgClient.Start(ctx); err != nil && err != context.Canceled {
			log.Fatal("Telegram client failed", zap.Error(err))
		}
	}()

	select {
	case <-tgClient.WaitReady():
		log.Info("Telegram client ready")
	case <-ctx.Done():
		log.Fatal("Shutdown before client ready")
	}

	svc := service.NewAnalyticsService(tgClient)
	hdl := rest.NewHandler(svc)
	router := rest.SetupRouter(hdl, log)

	srv := &http.Server{Addr: ":" + cfg.HTTPPort, Handler: router}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = tgClient.Stop(context.Background())
	log.Info("Exited")
}
