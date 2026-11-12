package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
)

// DBHandler wraps gorm.DB with additional functionality
type DBHandler struct {
	DB  *gorm.DB
	SQL *sql.DB
}

// InitializeDB connects to PostgreSQL using GORM
func InitializeDB(cfg *config.Config) (*DBHandler, error) {
	// If no DATABASE_URL provided, use a default local development URL
	databaseURL := cfg.DatabaseURL
	if databaseURL == "" {
		// Development default - production must set this environment variable.
		databaseURL = "postgres://localhost/edwinslab_development?sslmode=disable"
		slog.Warn("DATABASE_URL not set, using local development default")
	}

	// Open connection using gorm.io/driver/postgres
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * 60) // 30 minutes

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	slog.Info("Database connected successfully", "url_masked", maskDBURL(databaseURL))

	return &DBHandler{
		DB:  db,
		SQL: sqlDB,
	}, nil
}

// maskDBURL masks sensitive parts of a database URL for logging
func maskDBURL(url string) string {
	// Simple masking - replace password with ****
	if len(url) > 20 {
		return url[:10] + "****" + url[len(url)-10:]
	}
	return url
}

// CloseDB closes the database connection
func (h *DBHandler) CloseDB() error {
	// GORM will close the connection when the DB is garbage collected,
	// but we explicitly close the SQL connection
	return h.SQL.Close()
}

// Migrate runs all database migrations from the embedded SQL files.
func (h *DBHandler) Migrate() error {
	if err := h.DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		return fmt.Errorf("failed to create pgcrypto extension: %w", err)
	}

	if err := Migrate(h.DB, slog.Default()); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	slog.Info("Database migrations completed")
	return nil
}

// GetDB returns the underlying *gorm.DB
func (h *DBHandler) GetDB() *gorm.DB {
	return h.DB
}

// GetRawDB returns the underlying *sql.DB for raw queries
func (h *DBHandler) GetRawDB() *sql.DB {
	return h.SQL
}