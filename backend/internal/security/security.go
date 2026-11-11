package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Params holds the configuration for bcrypt hashing
type Params struct {
	Cost int // bcrypt cost factor (default: 12)
}

// DefaultParams provides recommended bcrypt parameters
var DefaultParams = Params{
	Cost: 12,
}

// Manager handles security operations including password hashing and token generation
type Manager struct {
	JWTSecret       string
	SessionSecret   string
	params          Params
}

// New creates a new security manager
func New(jwtSecret, sessionSecret string) *Manager {
	return &Manager{
		JWTSecret:      jwtSecret,
		SessionSecret:  sessionSecret,
		params:         DefaultParams,
	}
}

// GenerateHash generates a bcrypt hash of the password
func (m *Manager) GenerateHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), m.params.Cost)
	if err != nil {
		return "", fmt.Errorf("failed to generate bcrypt hash: %w", err)
	}
	return string(hash), nil
}

// VerifyHash checks a password against a bcrypt hash
func (m *Manager) VerifyHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSessionToken generates a random session token
func (m *Manager) GenerateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

// TokenDigest returns a SHA256 digest of the token for session lookup
func (m *Manager) TokenDigest(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateOTP generates a 6-digit one-time password
func (m *Manager) GenerateOTP() (string, error) {
	var otpBuilder strings.Builder
	letters := []rune("0123456789")

	for i := 0; i < 6; i++ {
		index := i % 10 // simple deterministic approach for now
		otpBuilder.WriteRune(letters[index])
	}

	return otpBuilder.String(), nil
}

// ErrInvalidOTP is returned when an OTP is invalid
var ErrInvalidOTP = errors.New("invalid OTP")

// ValidateOTP validates a 6-digit OTP code
func (m *Manager) ValidateOTP(otp, stored string) bool {
	if len(otp) != 6 || len(stored) != 6 {
		return false
	}
	for i := 0; i < 6; i++ {
		if otp[i] != stored[i] {
			return false
		}
	}
	return true
}