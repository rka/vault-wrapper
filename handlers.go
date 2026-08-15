package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
)

const (
	maxTTLSeconds  = 30 * 24 * 60 * 60
	maxTokenLength = 16 * 1024
	maxFileCount   = 100
)

type wrappedFile struct {
	IsFile bool   `json:"isFile,omitempty"`
	Name   string `json:"name"`
	Type   string `json:"type,omitempty"`
	Data   string `json:"data"`
	Size   int64  `json:"size"`
}

type wrapPayload struct {
	Text  string        `json:"text,omitempty"`
	Files []wrappedFile `json:"files,omitempty"`
}

type wrapRequest struct {
	Data wrapPayload `json:"data"`
	TTL  string      `json:"ttl"`
}

// getClientIP returns a stable address suitable for rate-limiter keys. Proxy
// headers are honored only when TRUST_PROXY_HEADERS is explicitly enabled.
func getClientIP(r *http.Request) string {
	if TrustProxyHeaders {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			candidate := strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// reqID returns the request ID injected by loggingMiddleware, or "-".
func reqID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return "-"
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "clipboard-read=(self), clipboard-write=(self), camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/static/") {
			// Asset names are stable rather than content-hashed, so clients must
			// revalidate to avoid mixing a new HTML shell with stale JS or CSS.
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func methodAllowed(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	return false
}

func requestBodyLimit() int64 {
	// Base64 expands binary attachments by roughly 4/3. A 2x envelope leaves
	// room for JSON metadata while the decoded payload is validated separately.
	if MaxRequestSize > (math.MaxInt64-64*1024)/2 {
		return math.MaxInt64
	}
	return MaxRequestSize*2 + 64*1024
}

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, destination any, limit int64) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("Content-Type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

func validatePayload(payload wrapPayload) (int64, error) {
	if payload.Text == "" && len(payload.Files) == 0 {
		return 0, errors.New("data must include text or at least one file")
	}
	if len(payload.Files) > maxFileCount {
		return 0, fmt.Errorf("a maximum of %d files is allowed", maxFileCount)
	}

	total := int64(len([]byte(payload.Text)))
	if total > MaxRequestSize {
		return total, fmt.Errorf("payload exceeds the maximum size of %d bytes", MaxRequestSize)
	}
	for _, file := range payload.Files {
		if strings.TrimSpace(file.Name) == "" || len(file.Name) > 255 {
			return 0, errors.New("each file must have a name no longer than 255 characters")
		}
		if len(file.Type) > 255 || file.Size < 0 {
			return 0, fmt.Errorf("file %q has invalid metadata", file.Name)
		}
		decoded := base64.NewDecoder(base64.StdEncoding.Strict(), strings.NewReader(file.Data))
		size, err := io.Copy(io.Discard, io.LimitReader(decoded, MaxRequestSize+1))
		if err != nil {
			return 0, fmt.Errorf("file %q is not valid base64 data", file.Name)
		}
		if size != file.Size {
			return 0, fmt.Errorf("file %q size does not match its contents", file.Name)
		}
		if total > MaxRequestSize-size {
			return total + size, fmt.Errorf("payload exceeds the maximum size of %d bytes", MaxRequestSize)
		}
		total += size
	}
	return total, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !methodAllowed(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	http.ServeFile(w, r, "./static/index.html")
}

func wrapHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}

	var input wrapRequest
	if err := decodeJSONRequest(w, r, &input, requestBodyLimit()); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "request body too large") {
			status = http.StatusRequestEntityTooLarge
		}
		log.Printf("WARN  [%s] wrap: invalid request: %v", reqID(r), err)
		http.Error(w, err.Error(), status)
		return
	}

	ttlValue, err := parseTTL(input.TTL)
	if err != nil {
		log.Printf("WARN  [%s] wrap: invalid TTL %q", reqID(r), input.TTL)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	input.TTL = fmt.Sprintf("%d", ttlValue)

	payloadSize, err := validatePayload(input.Data)
	if err != nil {
		log.Printf("WARN  [%s] wrap: invalid payload: %v", reqID(r), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dataBytes, err := json.Marshal(input.Data)
	if err != nil {
		log.Printf("ERROR [%s] wrap: failed to marshal data: %v", reqID(r), err)
		http.Error(w, "Could not prepare payload", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO  [%s] wrap: payload_size=%d ttl=%s ip=%s", reqID(r), payloadSize, input.TTL, getClientIP(r))
	token, details, err := wrapData(string(dataBytes), input.TTL)
	if err != nil {
		log.Printf("ERROR [%s] wrap: wrapData failed: %v", reqID(r), err)
		http.Error(w, "Vault could not wrap the payload", http.StatusBadGateway)
		return
	}

	log.Printf("INFO  [%s] wrap: success token=%s", reqID(r), maskToken(token))
	receipt := map[string]any{
		"accessor":         details.Accessor,
		"ttl":              details.TTL,
		"creation_time":    details.CreationTime,
		"creation_path":    details.CreationPath,
		"wrapped_accessor": details.WrappedAccessor,
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "details": receipt})
}

func parseTTL(value string) (int, error) {
	ttl, err := strconv.Atoi(value)
	if err != nil || ttl <= 0 {
		return 0, errors.New("TTL must be a positive integer")
	}
	if ttl > maxTTLSeconds {
		return 0, fmt.Errorf("TTL cannot exceed %d seconds (30 days)", maxTTLSeconds)
	}
	return ttl, nil
}

func unwrapHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodPost) {
		return
	}

	var input struct {
		Token string `json:"token"`
	}
	if err := decodeJSONRequest(w, r, &input, 32*1024); err != nil {
		log.Printf("WARN  [%s] unwrap: invalid request: %v", reqID(r), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	input.Token = strings.TrimSpace(input.Token)
	if input.Token == "" || len(input.Token) > maxTokenLength {
		http.Error(w, "Token is required and must be shorter than 16 KB", http.StatusBadRequest)
		return
	}

	masked := maskToken(input.Token)
	log.Printf("INFO  [%s] unwrap: token=%s ip=%s", reqID(r), masked, getClientIP(r))

	// Read metadata before consuming the one-time token.
	tokenInfo, lookupErr := lookupWrappingToken(input.Token)
	if lookupErr != nil {
		log.Printf("WARN  [%s] unwrap: lookup failed (token=%s): %v", reqID(r), masked, lookupErr)
	}

	dataMap, err := unwrapData(input.Token)
	if err != nil {
		log.Printf("WARN  [%s] unwrap: failed (token=%s): %v", reqID(r), masked, err)
		if isInvalidWrappingTokenError(err) {
			http.Error(w, "Token is invalid, expired, or already used", http.StatusNotFound)
			return
		}
		http.Error(w, "Vault could not unwrap the payload", http.StatusBadGateway)
		return
	}

	dataString, ok := dataMap["data"].(string)
	if !ok {
		log.Printf("ERROR [%s] unwrap: unexpected data shape", reqID(r))
		http.Error(w, "Vault returned an unexpected payload", http.StatusBadGateway)
		return
	}
	var data wrapPayload
	if err := json.Unmarshal([]byte(dataString), &data); err != nil {
		log.Printf("ERROR [%s] unwrap: failed to unmarshal payload: %v", reqID(r), err)
		http.Error(w, "Vault returned an unreadable payload", http.StatusBadGateway)
		return
	}

	log.Printf("INFO  [%s] unwrap: success token=%s", reqID(r), masked)
	response := map[string]any{"data": data}
	if tokenInfo != nil {
		response["wrapping_info"] = tokenInfo.Data
	}
	writeJSON(w, http.StatusOK, response)
}

func isInvalidWrappingTokenError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "wrapping token is not valid") ||
		strings.Contains(message, "wrapping token is invalid") ||
		strings.Contains(message, "does not exist")
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"version":          Version,
		"github_url":       GithubURL,
		"max_payload_size": MaxRequestSize,
		"max_request_size": MaxRequestSize, // Backward compatibility for older clients.
	})
}

func vaultHealthHandler(w http.ResponseWriter, r *http.Request) {
	if !methodAllowed(w, r, http.MethodGet, http.MethodHead) {
		return
	}
	health := getVaultHealth()
	status := http.StatusOK
	if health.Status == "unhealthy" {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, health)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("ERROR writing JSON response: %v", err)
	}
}

// maskToken returns a redacted form of a token safe for logs.
func maskToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
