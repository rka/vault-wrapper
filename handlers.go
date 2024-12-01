package main

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/index.html")
}

func wrapHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Handling wrap request")
	if r.Method != http.MethodPost {
		logrus.Warn("Method not allowed for wrap request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Text string `json:"text"`
		TTL  string `json:"ttl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		logrus.Error("Error decoding wrap request body: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := wrapData(data.Text, data.TTL)
	if err != nil {
		logrus.Error("Error wrapping data: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logrus.Debug("Data wrapped successfully")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func unwrapHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Handling unwrap request")
	if r.Method != http.MethodPost {
		logrus.Warn("Method not allowed for unwrap request")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		logrus.Error("Error decoding unwrap request body: ", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	unwrappedData, err := unwrapData(data.Token)
	if err != nil {
		logrus.Error("Error unwrapping data: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	logrus.Debug("Data unwrapped successfully")
	json.NewEncoder(w).Encode(map[string]string{"data": unwrappedData})
}