package main

import (
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
	"io/ioutil"
)

const vaultAddr = "http://vault:8200"
const vaultToken = "root"

func wrapData(data string) (string, error) {
	payload := map[string]string{"data": data}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", vaultAddr+"/v1/sys/wrapping/wrap", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", vaultToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vault returned non-200 status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	wrapInfo, ok := result["wrap_info"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	token, ok := wrapInfo["token"].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}

	return token, nil
}

func unwrapData(token string) (string, error) {
	req, err := http.NewRequest("POST", vaultAddr+"/v1/sys/wrapping/unwrap", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vault returned non-200 status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	unwrappedData, ok := data["data"].(string)
	if !ok {
		return "", fmt.Errorf("data not found in response")
	}

	return unwrappedData, nil
}