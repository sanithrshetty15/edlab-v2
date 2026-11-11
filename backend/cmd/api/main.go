package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/sanithrshetty15/edwinslab/backend/internal/api"
	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/database"
	"github.com/sanithrshetty15/edwinslab/backend/internal/jobs"
	"github.com/sanithrshetty15/edwinslab/backend/internal/mailer"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	log.Printf("Starting Edwins Lab API, environment: %s, port: %s", cfg.AppEnv, cfg.Port)

	dbHandler, err := database.InitializeDB(&cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer dbHandler.CloseDB()

	if err := dbHandler.Migrate(); err != nil {
		log.Fatal("Failed to run database migrations:", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	mail := mailer.New()
	jobManager := jobs.New(&cfg)

	server := api.New(dbHandler.DB, cfg, logger, mail, jobManager)

	// Use the server's router which includes all API routes plus health checks
	router := server.Router()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf(":%s", cfg.Port)
		log.Printf("Server starting on %s", addr)
		if err := router.Run(addr); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server error:", err)
		}
	}()

	<-stop
	log.Println("Shutting down server...")

	log.Println("Server stopped")
}