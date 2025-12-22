package main

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"time"
	"token-bucket/ratelimit"
)

func health(w http.ResponseWriter, req *http.Request) {
	log.Println(req.URL.Path, "service healthy")
	w.WriteHeader(http.StatusOK)
}

func heavyWithRateLimit(next http.Handler, limiter ratelimit.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		host, _, _ := net.SplitHostPort(req.RemoteAddr)
		allowResult := limiter.Allow(host)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(int(allowResult.Capacity)))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(int(allowResult.Remaining)))
		w.Header().Set("Retry-After", strconv.Itoa(int(allowResult.RetryAfter.Seconds())))
		if !allowResult.Allowed {
			http.Error(w, "Too Many Requests from "+req.RemoteAddr, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func heavy(w http.ResponseWriter, req *http.Request) {
	log.Println(req.URL.Path, "executing heavy handler")
	// simulate some work
	time.Sleep(100 * time.Millisecond)
	w.Write([]byte("OK"))
}

func main() {
	clock := &ratelimit.RealClock{}
	capacity := int32(1)
	limiter := ratelimit.NewTokenBucketLimiter(0.1, capacity, capacity, clock)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.Handle("/heavy", heavyWithRateLimit(http.HandlerFunc(heavy), limiter))
	http.ListenAndServe(":8090", mux)
}
