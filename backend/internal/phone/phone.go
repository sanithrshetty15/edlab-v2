package phone

import (
	"regexp"
	"strings"
)

var nonDigits = regexp.MustCompile(`[^0-9]`)

func Normalize(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	digits := nonDigits.ReplaceAllString(trimmed, "")
	switch {
	case len(digits) == 10:
		return "+91" + digits
	case len(digits) == 11 && strings.HasPrefix(digits, "0"):
		return "+91" + digits[1:]
	case len(digits) == 12 && strings.HasPrefix(digits, "91"):
		return "+" + digits
	case len(digits) == 13 && strings.HasPrefix(digits, "091"):
		return "+" + digits[1:]
	case strings.HasPrefix(trimmed, "+") && len(digits) >= 8:
		return "+" + digits
	default:
		return strings.TrimSpace(trimmed)
	}
}

func IsValid(value string) bool {
	normalized := Normalize(value)
	return len(normalized) >= 11 && strings.HasPrefix(normalized, "+")
}

func Last10(value string) string {
	digits := nonDigits.ReplaceAllString(value, "")
	if len(digits) > 10 {
		return digits[len(digits)-10:]
	}
	return digits
}
