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

    var input struct {
        Data interface{} `json:"data"`
        TTL  string      `json:"ttl"`
    }
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        logrus.Error("Error decoding wrap request body: ", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    dataBytes, err := json.Marshal(input.Data)
    if err != nil {
        logrus.Error("Error marshalling data: ", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    dataString := string(dataBytes)
    token, details, err := wrapData(dataString, input.TTL)
    if err != nil {
        logrus.Error("Error wrapping data: ", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    logrus.Debug("Data wrapped successfully")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "token":   token,
        "details": details,
    })
}

func unwrapHandler(w http.ResponseWriter, r *http.Request) {
    logrus.Debug("Handling unwrap request")
    if r.Method != http.MethodPost {
        logrus.Warn("Method not allowed for unwrap request")
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var input struct {
        Token string `json:"token"`
    }
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        logrus.Error("Error decoding unwrap request body: ", err)
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    dataMap, err := unwrapData(input.Token)
    if err != nil {
        logrus.Error("Error unwrapping data: ", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // The data is stored under the "data" key
    dataString, ok := dataMap["data"].(string)
    if !ok {
        logrus.Error("Error retrieving unwrapped data")
        http.Error(w, "Error retrieving unwrapped data", http.StatusInternalServerError)
        return
    }

    var data interface{}
    if err := json.Unmarshal([]byte(dataString), &data); err != nil {
        logrus.Error("Error unmarshalling data: ", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    logrus.Debug("Data unwrapped successfully")
    json.NewEncoder(w).Encode(data)
}
