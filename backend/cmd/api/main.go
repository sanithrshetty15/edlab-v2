package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sanithrshetty15/edwinslab/backend/internal/api"
	"github.com/sanithrshetty15/edwinslab/backend/internal/bootstrap"
	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/database"
	"github.com/sanithrshetty15/edwinslab/backend/internal/jobs"
	"github.com/sanithrshetty15/edwinslab/backend/internal/mailer"
	"github.com/sanithrshetty15/edwinslab/backend/internal/security"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	logger.Info("starting Edwins Lab API", "environment", cfg.AppEnv, "port", cfg.Port)

	dbHandler, err := database.InitializeDB(&cfg)
	if err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer dbHandler.CloseDB()

	if err := dbHandler.Migrate(); err != nil {
		logger.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}

	securityManager := security.New(cfg.JWTSecret, cfg.SessionSecret)
	ctx := context.Background()

	if err := bootstrap.EnsureRequiredAdmins(ctx, dbHandler.DB, logger, securityManager, cfg.BootstrapReset); err != nil {
		logger.Error("failed to provision required administrators", "error", err)
		os.Exit(1)
	}
	if err := bootstrap.SeedDefaults(ctx, dbHandler.DB, logger); err != nil {
		logger.Error("failed to seed platform defaults", "error", err)
		os.Exit(1)
	}

	mail := mailer.New()
	if cfg.MailerFrom != "" {
		mail.From = cfg.MailerFrom
	}
	if cfg.FrontendURL != "" {
		mail.FrontendURL = cfg.FrontendURL
	}
	if !mail.Configured() {
		logger.Warn("no email provider configured; verification and notification emails will fail")
	}

	jobManager := jobs.New(&cfg, dbHandler.DB, logger)
	server := api.New(dbHandler.DB, cfg, logger, mail, jobManager)

	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	jobManager.Start(workerCtx, 2)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server listening", "addr", httpServer.Addr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case signalValue := <-stop:
		logger.Info("shutdown signal received", "signal", signalValue.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
	cancelWorkers()
	jobManager.Wait()
	logger.Info("server stopped")
}
