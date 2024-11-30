package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

// const (
// 	vaultAddr  = "http://vault:8200" // Define the Vault address
// 	vaultToken = "root"               // Define the Vault token (consider using a safer method for production)
// )

// wrapData wraps the provided data using Vault's wrapping functionality.
func wrapData(data string) (string, error) {
	logrus.Debug("Wrapping data")
	payload := map[string]string{"data": data}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logrus.Error("Error marshalling JSON payload: ", err)
		return "", err
	}

	req, err := http.NewRequest("POST", vaultAddr+"/v1/sys/wrapping/wrap", bytes.NewBuffer(jsonPayload))
	if err != nil {
		logrus.Error("Error creating wrap request: ", err)
		return "", err
	}
	req.Header.Set("X-Vault-Token", vaultToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error("Error sending wrap request: ", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Warn("Vault returned non-200 status: ", resp.StatusCode)
		return "", fmt.Errorf("vault returned non-200 status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		logrus.Error("Error decoding response from Vault: ", err)
		return "", err
	}

	wrapInfo, ok := result["wrap_info"].(map[string]interface{})
	if !ok {
		logrus.Error("Unexpected response format from Vault")
		return "", fmt.Errorf("unexpected response format")
	}

	token, ok := wrapInfo["token"].(string)
	if !ok {
		logrus.Error("Token not found in Vault response")
		return "", fmt.Errorf("token not found in response")
	}

	logrus.Debug("Data wrapped successfully")
	return token, nil
}

// unwrapData unwraps the provided token using Vault's unwrapping functionality.
func unwrapData(token string) (string, error) {
	logrus.Debug("Unwrapping data")
	req, err := http.NewRequest("POST", vaultAddr+"/v1/sys/wrapping/unwrap", nil)
	if err != nil {
		logrus.Error("Error creating unwrap request: ", err)
		return "", err
	}
	req.Header.Set("X-Vault-Token", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error("Error sending unwrap request: ", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Warn("Vault returned non-200 status: ", resp.StatusCode)
		return "", fmt.Errorf("vault returned non-200 status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error("Error reading unwrap response body: ", err)
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logrus.Error("Error unmarshalling response body: ", err)
		return "", err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		logrus.Error("Unexpected response format from Vault")
		return "", fmt.Errorf("unexpected response format")
	}

	unwrappedData, ok := data["data"].(string)
	if !ok {
		logrus.Error("Data not found in Vault response")
		return "", fmt.Errorf("data not found in response")
	}

	logrus.Debug("Data unwrapped successfully")
	return unwrappedData, nil
}