package main

import (
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors   = make(map[string]*visitor)
	visitorsMu sync.Mutex
)

// cleanupVisitors removes idle per-IP limiter entries every minute.
// Should be started as a goroutine from main().
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)
		visitorsMu.Lock()
		for key, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, key)
			}
		}
		visitorsMu.Unlock()
	}
}

// getVisitor returns the rate.Limiter for the given key (typically ip+endpoint),
// creating one with the supplied limit and burst if it does not yet exist.
func getVisitor(key string, r rate.Limit, burst int) *rate.Limiter {
	visitorsMu.Lock()
	defer visitorsMu.Unlock()

	v, ok := visitors[key]
	if !ok {
		l := rate.NewLimiter(r, burst)
		visitors[key] = &visitor{limiter: l, lastSeen: time.Now()}
		return l
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// rateLimitMiddleware wraps h and enforces per-IP rate limiting with the
// given rate.Limit and burst.  Rejected requests receive HTTP 429.
func rateLimitMiddleware(h http.HandlerFunc, r rate.Limit, burst int, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ip := getClientIP(req)
		limiter := getVisitor(ip+":"+endpoint, r, burst)
		if !limiter.Allow() {
			log.Printf("WARN  rate limit exceeded — endpoint=%s ip=%s", endpoint, ip)
			http.Error(w, "Too many requests. Please slow down.", http.StatusTooManyRequests)
			return
		}
		h(w, req)
	}
}
