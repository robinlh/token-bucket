package ratelimit

import (
	"sync"
	"time"
)

type AllowResult struct {
	Allowed    bool
	Capacity   int32
	Remaining  int32
	RetryAfter time.Duration
}

type Limiter interface {
	Allow(key string) AllowResult
}

type TokenBucket struct {
	clock               Clock
	capacity            int32
	refillRate          float64
	tokens              int32
	tokenRemainder      float64
	lastRefillTimestamp time.Time
	mu                  sync.Mutex
}

type TokenBucketLimiter struct {
	mu       sync.RWMutex
	rate     float64
	capacity int32
	tokens   int32
	clock    Clock
	buckets  map[string]*TokenBucket
}

func (tb *TokenBucket) refill() {
	now := tb.clock.Now()
	elapsed := now.Sub(tb.lastRefillTimestamp)

	newTokensFloat := tb.refillRate*elapsed.Seconds() + tb.tokenRemainder
	newTokensInt := int32(newTokensFloat)
	tb.tokenRemainder = newTokensFloat - float64(newTokensInt)

	if newTokensInt > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+newTokensInt)
		tb.lastRefillTimestamp = now
	}
}

func (tbl *TokenBucketLimiter) getOrCreateBucket(key string) *TokenBucket {
	tbl.mu.Lock()
	defer tbl.mu.Unlock()

	tb, ok := tbl.buckets[key]
	if !ok {
		tb = NewTokenBucket(tbl.rate, tbl.capacity, tbl.tokens, tbl.clock)
		tbl.buckets[key] = tb
	}
	return tb
}

func (tbl *TokenBucketLimiter) Allow(key string) AllowResult {
	tbl.mu.RLock()
	tb, ok := tbl.buckets[key]
	tbl.mu.RUnlock()

	if !ok {
		tb = tbl.getOrCreateBucket(key)
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1 {
		tb.tokens -= 1
		return AllowResult{
			Allowed:    true,
			Capacity:   tb.capacity,
			Remaining:  tb.tokens,
			RetryAfter: 0,
		}
	} else {
		timeForOneToken := time.Duration(float64(time.Second) / tb.refillRate)
		return AllowResult{
			Allowed:    false,
			Capacity:   tb.capacity,
			Remaining:  tb.tokens,
			RetryAfter: timeForOneToken,
		}
	}
}

func NewTokenBucket(rate float64, capacity int32, tokens int32, clock Clock) *TokenBucket {
	return &TokenBucket{
		clock:               clock,
		capacity:            capacity,
		refillRate:          rate,
		tokens:              tokens,
		tokenRemainder:      0,
		lastRefillTimestamp: clock.Now(),
	}
}

func NewTokenBucketLimiter(rate float64, capacity int32, tokens int32, clock Clock) *TokenBucketLimiter {
	buckets := make(map[string]*TokenBucket)
	return &TokenBucketLimiter{buckets: buckets, rate: rate, capacity: capacity, tokens: tokens, clock: clock}
}
