package database

import (
	"database/sql"
	"fmt"
	"log/slog"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/sanithrshetty15/edwinslab/backend/internal/config"
	"github.com/sanithrshetty15/edwinslab/backend/internal/models"
)

// DBHandler wraps gorm.DB with additional functionality
type DBHandler struct {
	DB  *gorm.DB
	SQL *sql.DB
}

// InitializeDB connects to PostgreSQL using GORM
func InitializeDB(cfg *config.Config) (*DBHandler, error) {
	// If no DATABASE_URL provided, use a default local development URL
	databaseURL := cfg.DATABASEURL
	if databaseURL == "" {
		// Development default - production will set this env var
		databaseURL = "postgres://sanith:localhost@localhost:5432/edwins_lab_development?sslmode=disable"
		slog.Warn("DATABASE_URL not set, using development default")
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

// Migrate runs all database migrations
func (h *DBHandler) Migrate(value ...interface{}) error {
	// Enable pgcrypto for random UUID generation and password hashing
	err := h.DB.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error
	if err != nil {
		return fmt.Errorf("failed to create pgcrypto extension: %w", err)
	}

	// Run migrations
	err = h.DB.AutoMigrate(
		&models.User{},
		&models.Member{},
		&models.Admin{},
		&models.Session{},
		&models.OTPCode{},
		&models.BiometricEnrollment{},
		&models.DeviceRegistration{},
		&models.DeviceEvent{},
		&models.AttendanceLog{},
		&models.Announcement{},
		&models.AnnouncementRead{},
		&models.Notification{},
		&models.InventoryItem{},
		&models.BorrowRecord{},
		&models.ExtensionRequest{},
		&models.ReturnRequest{},
		&models.Task{},
		&models.TaskTarget{},
		&models.TaskSubmissionStatus{},
		&models.Event{},
		&models.FAQCategory{},
		&models.FAQ{},
		&models.ContactMessage{},
		&models.GroupMessage{},
		&models.MessageAttachment{},
		&models.MessageReceipt{},
		&models.MessageReaction{},
		&models.MessagePin{},
		&models.MessageStar{},
		&models.LinkPreview{},
		&models.Poll{},
		&models.PollOption{},
		&models.PollVote{},
		&models.AdminTransfer{},
		&models.LabRulesVersion{},
		&models.LabRulesAcceptance{},
		&models.PlatformSetting{},
		&models.AuditLog{},
		&models.ContentBlock{},
	)
	if err != nil {
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