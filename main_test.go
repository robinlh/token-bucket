package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"token-bucket/ratelimit"
)

func TestServer(t *testing.T) {
	clock := &ratelimit.MockClock{
		CurrentTime: time.Now(),
	}
	limiter := ratelimit.NewTokenBucketLimiter(2.0, 1, 1, clock)

	mux := http.NewServeMux()
	mux.Handle("/heavy", heavyWithRateLimit(http.HandlerFunc(heavy), limiter))

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/heavy")

	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	resp, err = http.Get(server.URL + "/heavy")

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", resp.StatusCode)
	}

	clock.Advance(500 * time.Millisecond)

	resp, err = http.Get(server.URL + "/heavy")

	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
