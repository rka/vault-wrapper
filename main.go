package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// Version is set during build
var Version = "dev"
var GithubURL = ""
var MaxRequestSize int64 = 5 * 1024 * 1024 // Default 5 MB

var requestCounter uint64

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	if url := os.Getenv("GITHUB_URL"); url != "" {
		GithubURL = url
	}

	if sizeStr := os.Getenv("MAX_REQUEST_SIZE"); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			MaxRequestSize = size
		} else {
			log.Printf("WARN  invalid MAX_REQUEST_SIZE: %v — using default %d bytes", err, MaxRequestSize)
		}
	}

	if data, err := os.ReadFile("version.txt"); err == nil {
		Version = strings.TrimSpace(string(data))
	}

	log.Printf("Starting Vault Data Wrapper v%s", Version)
}

// nextReqID returns a monotonically incrementing request ID string.
func nextReqID() string {
	return fmt.Sprintf("%05d", atomic.AddUint64(&requestCounter, 1))
}

// loggingMiddleware logs the start and end of every HTTP request.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := nextReqID()
		start := time.Now()
		ip := getClientIP(r)
		log.Printf("INFO  [%s] --> %s %s ip=%s ua=%q", id, r.Method, r.URL.Path, ip, r.UserAgent())

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		// Inject request ID so handlers can reference it.
		r.Header.Set("X-Request-ID", id)
		h(rec, r)

		log.Printf("INFO  [%s] <-- %d (%s)", id, rec.status, time.Since(start).Round(time.Microsecond))
	}
}

func checkVaultConnectivity() error {
	log.Println("INFO  checking Vault connectivity...")
	health, err := vaultClient.Sys().Health()
	if err != nil {
		return fmt.Errorf("failed to reach Vault: %w", err)
	}
	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}
	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}
	log.Printf("INFO  Vault connectivity OK (version=%s cluster=%s)", health.Version, health.ClusterName)
	return nil
}

func main() {
	// Initialise the shared Vault client before anything else.
	if err := initVaultClient(); err != nil {
		log.Fatalf("FATAL failed to initialise Vault client: %v", err)
	}

	// Check Vault connectivity with retries.
	for i := 0; i < 5; i++ {
		if err := checkVaultConnectivity(); err != nil {
			log.Printf("WARN  Vault connectivity check failed (attempt %d/5): %v — retrying in 5s", i+1, err)
			time.Sleep(5 * time.Second)
		} else {
			break
		}
	}

	// Start background cleanup of idle rate-limiter entries.
	go cleanupVisitors()

	// Rate limits:  wrap = 10 req/min per IP (burst 5)
	//              unwrap = 30 req/min per IP (burst 10)
	wrapRate := rate.Every(6 * time.Second)   // 10/min
	unwrapRate := rate.Every(2 * time.Second) // 30/min

	// Apply logging + rate-limiting middleware to API endpoints.
	http.HandleFunc("/", loggingMiddleware(indexHandler))
	http.HandleFunc("/wrap", loggingMiddleware(
		rateLimitMiddleware(wrapHandler, wrapRate, 5, "wrap")))
	http.HandleFunc("/unwrap", loggingMiddleware(
		rateLimitMiddleware(unwrapHandler, unwrapRate, 10, "unwrap")))
	http.HandleFunc("/api/version", loggingMiddleware(versionHandler))
	http.HandleFunc("/api/health", loggingMiddleware(vaultHealthHandler))

	// Serve static files (no logging middleware — high frequency, low value)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("INFO  server listening on :3001")
	if err := http.ListenAndServe(":3001", nil); err != nil {
		log.Fatal(err)
	}
}
