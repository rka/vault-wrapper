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
// 	vaultAddr  = "http://vault:8200"
// 	vaultToken = "root"
// )

func wrapData(data string, ttl string) (string, error) {
	logrus.Debug("Wrapping data")

	payload := map[string]interface{}{"data": data}
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vault-Wrap-TTL", ttl)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Error("Error sending wrap request: ", err)
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	logrus.Infof("Vault response for wrap request: Status: %d, Body: %s", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
		logrus.Warnf("Vault returned non-200 status: %d", resp.StatusCode)
		return "", fmt.Errorf("vault returned non-200 status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
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

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	logrus.Infof("Vault response for unwrap request: Status: %d, Body: %s", resp.StatusCode, string(bodyBytes))

	if resp.StatusCode != http.StatusOK {
        logrus.Warnf("Vault returned non-200 status: %d", resp.StatusCode)
        return "", fmt.Errorf("vault returned non-200 status: %d", resp.StatusCode)
    }

	var result map[string]interface{}
	
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
	    logrus.Error("Error decoding response from Vault: ", err)
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