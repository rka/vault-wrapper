package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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

	dataString := string(dataBytes)
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

	dataMap, err := unwrapData(input.Token)
	if err != nil {
		log.Println("Error unwrapping data:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// The data is stored under the "data" key
	dataString, ok := dataMap["data"].(string)
	if !ok {
		log.Println("Error retrieving unwrapped data")
		http.Error(w, "Error retrieving unwrapped data",
			http.StatusInternalServerError)
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(dataString), &data); err != nil {
		log.Println("Error unmarshalling data:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Data unwrapped successfully")
	json.NewEncoder(w).Encode(data)
}
