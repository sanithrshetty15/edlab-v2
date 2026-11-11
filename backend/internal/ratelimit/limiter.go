package ratelimit

import (
	"sync"
	"time"
)

// Limiter provides simple in-memory rate limiting with token bucket algorithm
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
}

// tokenBucket represents a token bucket for a specific key
type tokenBucket struct {
	capacity  int64     // maximum tokens
	tokens    int64     // current token count
	lastRefill time.Time // last refill time
}

// New creates a new rate limiter
func New() *Limiter {
	return &Limiter{
		buckets: make(map[string]*tokenBucket),
	}
}

// Allow checks if a request is allowed for the given key
func (l *Limiter) Allow(key string, limit int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, exists := l.buckets[key]
	if !exists {
		bucket = &tokenBucket{
			capacity:  int64(limit),
			tokens:    int64(limit),
			lastRefill: time.Now(),
		}
		l.buckets[key] = bucket
	}

	// Refill tokens based on elapsed time
	elapsed := time.Since(bucket.lastRefill)
	refillCount := int64(elapsed / window)
	if refillCount > 0 {
		bucket.tokens = min(bucket.capacity, bucket.tokens+refillCount)
		bucket.lastRefill = bucket.lastRefill.Add(time.Duration(refillCount) * window)
	}

	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// min returns the minimum of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}