package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestAllow(t *testing.T) {
	// given
	mockClock := &MockClock{
		CurrentTime: time.Now(),
	}
	tbl := NewTokenBucketLimiter(5.0, 2, 2, mockClock)

	// when
	firstAllowed := tbl.Allow("key1")
	secondAllowed := tbl.Allow("key1")
	thirdNotAllowed := tbl.Allow("key1")

	// then
	if !firstAllowed.Allowed {
		t.Errorf("First allowed should be true but was false")
	}
	if !secondAllowed.Allowed {
		t.Errorf("Second allowed should be true but was false")
	}
	if thirdNotAllowed.Allowed {
		t.Errorf("Third not allowed should be false but was true")
	}
}

func TestRefill(t *testing.T) {
	tests := []struct {
		name           string
		rate           float64
		capacity       int32
		initialTokens  int32
		timeAdvance    time.Duration
		expectedTokens int32
		expectAllowed  bool
	}{
		{
			name:           "basic refill after 500ms",
			rate:           5.0,
			capacity:       4,
			initialTokens:  0,
			timeAdvance:    500 * time.Millisecond,
			expectedTokens: 1, // after Allow() consumes one
			expectAllowed:  true,
		},
		{
			name:           "fractional refill after 600ms",
			rate:           5.0,
			capacity:       4,
			initialTokens:  0,
			timeAdvance:    600 * time.Millisecond,
			expectedTokens: 2, // after Allow() consumes one
			expectAllowed:  true,
		},
		{
			name:           "fractional refill whole token after 1000ms",
			rate:           5.0,
			capacity:       5,
			initialTokens:  0,
			timeAdvance:    1000 * time.Millisecond,
			expectedTokens: 4, // after Allow() consumes one
			expectAllowed:  true,
		},
		{
			name:           "no refill after 0ms",
			rate:           5.0,
			capacity:       5,
			initialTokens:  0,
			timeAdvance:    0 * time.Millisecond,
			expectedTokens: 0, // not allowed
			expectAllowed:  false,
		},
		{
			name:           "no refill at capacity after 500ms",
			rate:           5.0,
			capacity:       5,
			initialTokens:  5,
			timeAdvance:    500 * time.Millisecond,
			expectedTokens: 4, // after Allow() consumes one
			expectAllowed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClock := &MockClock{CurrentTime: time.Now()}
			tbl := NewTokenBucketLimiter(tt.rate, tt.capacity, tt.initialTokens, mockClock)
			tbl.Allow("key1")
			mockClock.Advance(tt.timeAdvance)
			allowed := tbl.Allow("key1")

			if allowed.Allowed != tt.expectAllowed {
				t.Errorf("Should be allowed but wasn't")
			}

			tb, _ := tbl.buckets["key1"]
			if tb.tokens != tt.expectedTokens {
				t.Errorf("Expected %d token, got %d", tt.expectedTokens, tb.tokens)
			}
		})
	}
}

func TestMultiTenancy(t *testing.T) {
	clock := &RealClock{}
	tbl := NewTokenBucketLimiter(5.0, 2, 2, clock)

	tbl.Allow("key1")
	tbl.Allow("key1")
	tbl.Allow("key2")

	if tbl.Allow("key1").Allowed {
		t.Error("key1 should be exhausted")
	}
	if !tbl.Allow("key2").Allowed {
		t.Error("key2 should still have tokens")
	}
}

func TestConcurrentAccess(t *testing.T) {
	clock := &RealClock{}
	tbl := NewTokenBucketLimiter(100.0, 100, 100, clock)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tbl.Allow("key1")
			}
		}()
	}
	wg.Wait()
}
