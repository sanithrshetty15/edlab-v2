package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	JWTSecret      string
	SessionSecret  string
	Port           string
	FrontendURL    string
	CORSAllowedOrigins []string
	DATABASEURL    string
	SessionTTLMember time.Duration
	SessionTTLAdmin  time.Duration
	// OTP/SMS/Email config will be added when providers are configured
}

func LoadConfig() (Config, error) {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("error loading .env: %w", err)
	}

	cfg := Config{
		AppEnv: getEnv("APP_ENV", "development"),
		JWTSecret: getEnv("JWT_SECRET", ""),
		SessionSecret: getEnv("SESSION_SECRET", ""),
		Port: getEnv("PORT", "8080"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		CORSAllowedOrigins: parseCORSOrigins(getEnv("CORS_ALLOWED_ORIGINS", "")),
		DATABASEURL: getEnv("DATABASE_URL", ""),
		SessionTTLMember: 60 * time.Minute,
		SessionTTLAdmin:   24 * time.Hour,
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func parseCORSOrigins(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	origins := strings.Split(value, ",")
	result := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			result = append(result, o)
		}
	}
	return result
}

// Security returns a securely masked JWT secret for logging
func (c *Config) Security() string {
	if c.JWTSecret == "" {
		return "<not set>"
	}
	if len(c.JWTSecret) > 8 {
		return c.JWTSecret[:8] + "..." + c.JWTSecret[len(c.JWTSecret)-4:]
	}
	return "<short>"
}