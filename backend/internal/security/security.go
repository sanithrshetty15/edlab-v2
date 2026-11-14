package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// ArgonParams controls Argon2id hashing parameters.
type ArgonParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams provides strong Argon2id parameters for password hashing.
var DefaultParams = ArgonParams{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// Manager handles security operations including password hashing and token generation.
type Manager struct {
	JWTSecret     string
	SessionSecret string
	params        ArgonParams
}

// New creates a new security manager.
func New(jwtSecret, sessionSecret string) *Manager {
	return &Manager{
		JWTSecret:     jwtSecret,
		SessionSecret: sessionSecret,
		params:        DefaultParams,
	}
}

// GenerateHash generates an Argon2id hash of the password.
func (m *Manager) GenerateHash(password string) (string, error) {
	salt := make([]byte, m.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		m.params.Iterations,
		m.params.Memory,
		m.params.Parallelism,
		m.params.KeyLength,
	)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		m.params.Memory,
		m.params.Iterations,
		m.params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyHash checks a password against an Argon2id hash, falling back to legacy bcrypt.
func (m *Manager) VerifyHash(password, hash string) bool {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return false
	}
	if strings.HasPrefix(hash, "$argon2id$") {
		return verifyArgon2id(password, hash)
	}
	if strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") || strings.HasPrefix(hash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	return false
}

func verifyArgon2id(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}
	if memory == 0 || iterations == 0 || parallelism == 0 || memory > 1024*1024 {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return false
	}

	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

// GenerateSessionToken generates a random session token.
func (m *Manager) GenerateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

// TokenDigest returns a SHA256 digest of the token for session lookup.
func (m *Manager) TokenDigest(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateOTP generates a cryptographically secure four-digit numeric one-time
// password. Leading zeroes are preserved.
func (m *Manager) GenerateOTP() (string, error) {
	const digits = 4
	maximum := uint64(10000)
	value, err := randUint64(maximum)
	if err != nil {
		return "", fmt.Errorf("failed to generate OTP: %w", err)
	}
	return fmt.Sprintf("%0*d", digits, value), nil
}

func randUint64(max uint64) (uint64, error) {
	if max == 0 {
		return 0, errors.New("invalid random range")
	}
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return 0, err
	}
	value := binary.BigEndian.Uint64(buffer[:])
	limit := ^uint64(0) - (^uint64(0) % max)
	for value >= limit {
		if _, err := rand.Read(buffer[:]); err != nil {
			return 0, err
		}
		value = binary.BigEndian.Uint64(buffer[:])
	}
	return value % max, nil
}

// DigestOTP returns an HMAC digest for storing an OTP without keeping plaintext.
func (m *Manager) DigestOTP(code string) string {
	mac := hmac.New(sha256.New, []byte(m.SessionSecret))
	mac.Write([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

// IsOTPFormat reports whether code is exactly four numeric digits.
func IsOTPFormat(code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 4 {
		return false
	}
	for _, character := range code {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

// ValidateOTP compares a submitted OTP against a stored digest in constant time.
func (m *Manager) ValidateOTP(code, digest string) bool {
	if !IsOTPFormat(code) {
		return false
	}
	actual := m.DigestOTP(code)
	return hmac.Equal([]byte(actual), []byte(digest))
}

// ErrInvalidOTP is returned when an OTP is invalid.
var ErrInvalidOTP = errors.New("invalid OTP")

// PasswordResetRateLimited indicates the OTP request window was exhausted.
var ErrPasswordResetRateLimited = errors.New("too many verification code requests")

// ParsePositiveInt converts a string to a positive int64.
func ParsePositiveInt(value string) (int64, bool) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}
