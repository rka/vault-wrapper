package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func wrapHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling wrap request")
	if r.Method != http.MethodPost {
		log.Println("Method not allowed for wrap request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Data interface{} `json:"data"`
		TTL  string      `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Println("Error decoding wrap request body:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate TTL
	ttlValue, err := strconv.Atoi(input.TTL)
	if err != nil || ttlValue <= 0 {
		log.Println("Invalid TTL value:", input.TTL)
		http.Error(w, "Invalid TTL value. Must be a positive integer.",
			http.StatusBadRequest)
		return
	}

	dataBytes, err := json.Marshal(input.Data)
	if err != nil {
		log.Println("Error marshalling data:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get client IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	if strings.Contains(clientIP, ",") {
		clientIP = strings.Split(clientIP, ",")[0]
	}

	dataString := string(dataBytes)
	log.Printf("Wrapping data. Size: %d bytes, TTL: %s, IP: %s, User-Agent: %s",
		len(dataString), input.TTL, clientIP, r.UserAgent())

	token, details, err := wrapData(dataString, input.TTL)
	if err != nil {
		log.Println("Error wrapping data:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Data wrapped successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":   token,
		"details": details,
	})
}

func unwrapHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling unwrap request")
	if r.Method != http.MethodPost {
		log.Println("Method not allowed for unwrap request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		log.Println("Error decoding unwrap request body:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// First try to lookup the wrapping token
	tokenInfo, err := lookupWrappingToken(input.Token)
	if err != nil {
		log.Printf("Error looking up wrapping token: %v", err)
		// Don't return error here, continue with unwrap attempt
	}

	// Try to unwrap the data
	dataMap, err := unwrapData(input.Token)
	if err != nil {
		if tokenInfo != nil {
			// Token exists but can't be unwrapped - probably already used
			log.Printf("Token exists but cannot be unwrapped: %v", err)
			http.Error(w, fmt.Sprintf("Token details: Creation Time: %v, TTL: %v. Token has already been used or is invalid.",
				tokenInfo.Data["creation_time"], tokenInfo.Data["ttl"]), http.StatusBadRequest)
			return
		}

		// Handle other unwrap errors
		log.Printf("Error unwrapping data: %v", err)
		if strings.Contains(err.Error(), "wrapping token is not valid or does not exist") {
			http.Error(w, "Token is not valid or does not exist", http.StatusNotFound)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// The data is stored under the "data" key
	dataString, ok := dataMap["data"].(string)
	if !ok {
		log.Printf("Error retrieving unwrapped data, data map: %+v", dataMap)
		http.Error(w, "Error retrieving unwrapped data", http.StatusInternalServerError)
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(dataString), &data); err != nil {
		log.Printf("Error unmarshalling data: %v, data string: %s", err, dataString)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Include wrapping token info in the response if available
	response := map[string]interface{}{
		"data": data.(map[string]interface{}), // Extract the actual data structure
	}
	if tokenInfo != nil {
		response["wrapping_info"] = tokenInfo.Data
	}

	// Get client IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	if strings.Contains(clientIP, ",") {
		clientIP = strings.Split(clientIP, ",")[0]
	}

	// Mask token for logging
	maskedToken := "short-token"
	if len(input.Token) > 12 {
		maskedToken = input.Token[:4] + "..." + input.Token[len(input.Token)-4:]
	}

	log.Printf("Data unwrapped successfully. Token: %s, IP: %s, User-Agent: %s",
		maskedToken, clientIP, r.UserAgent())
	json.NewEncoder(w).Encode(response)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version":    Version,
		"github_url": GithubURL,
	})
}
