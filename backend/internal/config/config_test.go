package config

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestLoadConfigRejectsUnknownMailProvider(t *testing.T) {
	t.Setenv("MAIL_PROVIDER", "unknown")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig accepted an unknown MAIL_PROVIDER")
	}
}

func TestLoadConfigRequiresGmailOAuthConfiguration(t *testing.T) {
	t.Setenv("MAIL_PROVIDER", "gmail_api")
	t.Setenv("GMAIL_OAUTH_CLIENT_ID", "")
	t.Setenv("GMAIL_OAUTH_CLIENT_SECRET", "")
	t.Setenv("GMAIL_OAUTH_REDIRECT_URL", "")
	t.Setenv("GMAIL_TOKEN_ENCRYPTION_KEY", "")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig accepted gmail_api without OAuth configuration")
	}
}

func TestLoadConfigValidatesGmailEncryptionKey(t *testing.T) {
	t.Setenv("MAIL_PROVIDER", "gmail_api")
	t.Setenv("GMAIL_OAUTH_CLIENT_ID", "gmail-client")
	t.Setenv("GMAIL_OAUTH_CLIENT_SECRET", "gmail-secret")
	t.Setenv("GMAIL_OAUTH_REDIRECT_URL", "https://example.test/api/v1/oauth/gmail/callback")
	t.Setenv("GMAIL_TOKEN_ENCRYPTION_KEY", "short")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("LoadConfig accepted an invalid Gmail encryption key")
	}

	validKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0x42}, 32))
	t.Setenv("GMAIL_TOKEN_ENCRYPTION_KEY", validKey)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.MailProvider != "gmail_api" || cfg.GmailOAuthClientID != "gmail-client" {
		t.Fatalf("unexpected Gmail config: %+v", cfg)
	}
}
