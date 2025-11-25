package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vedsatt/pr-review-assignment-service/internal/config"
	"github.com/vedsatt/pr-review-assignment-service/internal/repository"
	"github.com/vedsatt/pr-review-assignment-service/internal/service"
	"github.com/vedsatt/pr-review-assignment-service/internal/transport"
	"go.uber.org/zap"
)

type App struct {
	Server     *http.Server
	Repository *repository.Repository
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	app := &App{}

	cfg, err := config.NewConfig()
	if err != nil {
		zap.L().Fatal("failed to get config: %v", zap.Error(err))
	}

	repository, err := repository.NewRepository(cfg.PostgresCfg)
	if err != nil {
		zap.L().Fatal("failed to create repository", zap.Error(err))
	}
	app.Repository = repository

	service := service.NewService(repository)

	zap.L().Info("starting server...", zap.String("port", cfg.HTTPPort))
	server := transport.StartServer(cfg, service)
	app.Server = server

	app.gracefulShutdown()
}

func (app *App) gracefulShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	<-quit
	zap.L().Info("shutdown signal received")

	const defaultShutdownTTL = time.Second * 10
	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTTL)
	defer cancel()

	zap.L().Info("shutting down HTTP server...")
	if err := app.Server.Shutdown(shutdownCtx); err != nil {
		zap.L().Error("failed to shutdown HTTP server", zap.Error(err))
	}

	zap.L().Info("closing database connection...")
	app.Repository.CloseConnection()

	zap.L().Info("app shotdown completed")
}
