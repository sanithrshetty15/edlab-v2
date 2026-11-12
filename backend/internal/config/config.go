package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv             string
	Port               string
	DatabaseURL        string
	JWTSecret          string
	SessionSecret      string
	FrontendURL        string
	CORSAllowedOrigins []string
	SessionTTLMember   time.Duration
	SessionTTLAdmin    time.Duration
	OTPExpiry          time.Duration
	OTPMaxAttempts     int
	OTPRequestWindow   time.Duration
	OTPMaxRequests     int
	OTPTestRecipient   string
	MailerFrom         string
	SMTPHost           string
	SMTPPort           string
	UploadDir          string
	MaxUploadBytes     int64
	RateLimitRequests  int
	RateLimitPeriod    time.Duration
	DeviceHMACSecret   string
	BootstrapReset     bool
	Timezone           string
}

func LoadConfig() (Config, error) {
	if err := loadEnvFiles(); err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),
		SessionSecret:      getEnv("SESSION_SECRET", ""),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		CORSAllowedOrigins: parseList(getEnv("CORS_ALLOWED_ORIGINS", getEnv("CORS_ORIGINS", ""))),
		SessionTTLMember:   durationEnv("SESSION_TTL_SECONDS", 24*time.Hour),
		SessionTTLAdmin:    durationEnv("ADMIN_SESSION_TTL_SECONDS", durationEnv("SESSION_TTL_SECONDS", 24*time.Hour)),
		OTPExpiry:          durationEnv("OTP_EXPIRY_SECONDS", 10*time.Minute),
		OTPMaxAttempts:     intEnv("OTP_MAX_ATTEMPTS", 5),
		OTPRequestWindow:   durationEnv("OTP_REQUEST_WINDOW_SECONDS", 15*time.Minute),
		OTPMaxRequests:     intEnv("OTP_MAX_REQUESTS", 5),
		OTPTestRecipient:   strings.TrimSpace(os.Getenv("OTP_TEST_RECIPIENT")),
		MailerFrom:         getEnv("MAILER_FROM", ""),
		SMTPHost:           getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:           getEnv("SMTP_PORT", "587"),
		UploadDir:          getEnv("UPLOAD_DIR", "data/uploads"),
		MaxUploadBytes:     int64(intEnv("MAX_UPLOAD_BYTES", 20*1024*1024)),
		RateLimitRequests:  intEnv("RATE_LIMIT_REQUESTS", 100),
		RateLimitPeriod:    durationEnv("RATE_LIMIT_PERIOD_SECONDS", time.Minute),
		DeviceHMACSecret:   getEnv("BIOMETRIC_API_SECRET", ""),
		BootstrapReset:     boolEnv("ADMIN_BOOTSTRAP_RESET_PASSWORDS", false),
		Timezone:           getEnv("APP_TIMEZONE", "Asia/Kolkata"),
	}

	if cfg.AppEnv == "production" {
		if cfg.DatabaseURL == "" {
			return Config{}, fmt.Errorf("DATABASE_URL must be set in production")
		}
		if len(cfg.SessionSecret) < 32 {
			return Config{}, fmt.Errorf("SESSION_SECRET must be at least 32 characters in production")
		}
		if len(cfg.JWTSecret) < 32 {
			return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
		}
	}

	if cfg.SessionSecret == "" {
		cfg.SessionSecret = randomSecret()
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = randomSecret()
	}
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = "postgres://localhost/edwinslab_development?sslmode=disable"
	}
	if cfg.MailerFrom == "" {
		if username := strings.TrimSpace(os.Getenv("GMAIL_USERNAME")); username != "" {
			cfg.MailerFrom = fmt.Sprintf("Edwins Lab <%s>", username)
		} else {
			cfg.MailerFrom = "Edwins Lab <noreply@edwinslab.in>"
		}
	}
	if cfg.OTPMaxAttempts < 1 {
		cfg.OTPMaxAttempts = 5
	}
	if cfg.OTPMaxRequests < 1 {
		cfg.OTPMaxRequests = 5
	}

	if len(cfg.CORSAllowedOrigins) == 0 {
		cfg.CORSAllowedOrigins = []string{
			"http://localhost:5173",
			"http://localhost:3000",
			"http://127.0.0.1:5173",
		}
	}
	cfg.CORSAllowedOrigins = mergeUnique(cfg.CORSAllowedOrigins, []string{
		"http://localhost",
		"https://localhost",
		"capacitor://localhost",
	})
	if cfg.FrontendURL != "" {
		cfg.CORSAllowedOrigins = mergeUnique(cfg.CORSAllowedOrigins, []string{cfg.FrontendURL})
	}

	return cfg, nil
}

func loadEnvFiles() error {
	candidates := []string{".env", filepath.Join("backend", ".env")}
	if executable, err := os.Executable(); err == nil {
		dir := filepath.Dir(executable)
		candidates = append(candidates,
			filepath.Join(dir, ".env"),
			filepath.Join(dir, "..", ".env"),
		)
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := godotenv.Load(path); err != nil {
			return fmt.Errorf("error loading %s: %w", path, err)
		}
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return defaultValue
}

func intEnv(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func boolEnv(key string, defaultValue bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func durationEnv(key string, defaultValue time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			return defaultValue
		}
		return time.Duration(seconds) * time.Second
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return defaultValue
	}
	return parsed
}

func parseList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func mergeUnique(base, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	result := make([]string, 0, len(base)+len(extra))
	for _, list := range [][]string{base, extra} {
		for _, item := range list {
			if item == "" {
				continue
			}
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func randomSecret() string {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("insecure-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}
