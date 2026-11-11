package api

import (
	"math"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

func dateValue(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(dateLayout)
}

func datePtr(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.Format(dateLayout)
}

func timeValue(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("15:04")
}

func clockValue(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("15:04:05")
}

func dateTimePtr(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value
}

func ptrString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func ptrInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func percentage(part, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(part)/float64(total)*10000) / 100
}
