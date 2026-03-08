package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// getClientIP extracts the real client IP, preferring the leftmost
// value in X-Forwarded-For (set by a trusted reverse proxy).
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.SplitN(xff, ",", 2)[0])
	}
	return r.RemoteAddr
}

// reqID returns the request ID injected by loggingMiddleware, or "-".
func reqID(r *http.Request) string {
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return "-"
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func wrapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enforce server-side body size limit.
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestSize)

	var input struct {
		Data interface{} `json:"data"`
		TTL  string      `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			log.Printf("WARN  [%s] wrap: body exceeds max size (%d bytes)", reqID(r), MaxRequestSize)
			http.Error(w, fmt.Sprintf("Request body exceeds maximum allowed size of %d bytes.", MaxRequestSize), http.StatusRequestEntityTooLarge)
			return
		}
		log.Printf("WARN  [%s] wrap: failed to decode body: %v", reqID(r), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ttlValue, err := strconv.Atoi(input.TTL)
	if err != nil || ttlValue <= 0 {
		log.Printf("WARN  [%s] wrap: invalid TTL %q", reqID(r), input.TTL)
		http.Error(w, "Invalid TTL value. Must be a positive integer.", http.StatusBadRequest)
		return
	}

	dataBytes, err := json.Marshal(input.Data)
	if err != nil {
		log.Printf("ERROR [%s] wrap: failed to marshal data: %v", reqID(r), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dataString := string(dataBytes)
	log.Printf("INFO  [%s] wrap: size=%d ttl=%s ip=%s", reqID(r), len(dataString), input.TTL, getClientIP(r))

	token, details, err := wrapData(dataString, input.TTL)
	if err != nil {
		log.Printf("ERROR [%s] wrap: wrapData failed: %v", reqID(r), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("INFO  [%s] wrap: success token=%s", reqID(r), maskToken(token))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"details": details,
	})
}

func unwrapHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Printf("WARN  [%s] unwrap: failed to decode body: %v", reqID(r), err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	masked := maskToken(input.Token)
	log.Printf("INFO  [%s] unwrap: token=%s ip=%s", reqID(r), masked, getClientIP(r))

	// Look up wrapping token metadata first (non-destructive).
	tokenInfo, err := lookupWrappingToken(input.Token)
	if err != nil {
		log.Printf("WARN  [%s] unwrap: lookup failed (token=%s): %v", reqID(r), masked, err)
		// Continue — the unwrap attempt below will surface the real error.
	}

	dataMap, err := unwrapData(input.Token)
	if err != nil {
		if tokenInfo != nil {
			log.Printf("WARN  [%s] unwrap: token exists but already consumed (token=%s): %v", reqID(r), masked, err)
			http.Error(w, fmt.Sprintf("Token has already been used. Details — creation_time: %v, ttl: %v",
				tokenInfo.Data["creation_time"], tokenInfo.Data["ttl"]), http.StatusBadRequest)
			return
		}
		log.Printf("WARN  [%s] unwrap: invalid token (token=%s): %v", reqID(r), masked, err)
		if strings.Contains(err.Error(), "wrapping token is not valid or does not exist") {
			http.Error(w, "Token is not valid or does not exist", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dataString, ok := dataMap["data"].(string)
	if !ok {
		log.Printf("ERROR [%s] unwrap: unexpected data shape: %+v", reqID(r), dataMap)
		http.Error(w, "Error retrieving unwrapped data", http.StatusInternalServerError)
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(dataString), &data); err != nil {
		log.Printf("ERROR [%s] unwrap: failed to unmarshal payload: %v", reqID(r), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dataMap2, ok := data.(map[string]interface{})
	if !ok {
		log.Printf("ERROR [%s] unwrap: payload is not a JSON object", reqID(r))
		http.Error(w, "Unexpected payload format", http.StatusInternalServerError)
		return
	}

	log.Printf("INFO  [%s] unwrap: success token=%s", reqID(r), masked)

	response := map[string]interface{}{"data": dataMap2}
	if tokenInfo != nil {
		response["wrapping_info"] = tokenInfo.Data
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":          Version,
		"github_url":       GithubURL,
		"max_request_size": MaxRequestSize,
	})
}

func vaultHealthHandler(w http.ResponseWriter, r *http.Request) {
	health := getVaultHealth()
	w.Header().Set("Content-Type", "application/json")
	// Use 200 for healthy/standby, 503 for unhealthy so monitoring tools work.
	if health.Status == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(health)
}

// maskToken returns a redacted form of a token safe for logs.
func maskToken(token string) string {
	if len(token) <= 12 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}
