package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/sirupsen/logrus"
)

var (
	vaultAddr  = os.Getenv("VAULT_ADDR")  // Get Vault address from environment
	vaultToken = os.Getenv("VAULT_TOKEN") // Get Vault token from environment
)

func init() {
	if vaultAddr == "" {
		vaultAddr = "http://vault:8200" // Default value if not set
		logrus.Warn("VAULT_ADDR not set, using default value: http://vault:8200")
	}
	if vaultToken == "" {
		vaultToken = "root" // Default value if not set
		logrus.Warn("VAULT_TOKEN not set, using default value: root")
	}
}

func wrapData(data string, ttl string) (string, *api.SecretWrapInfo, error) {
	client, err := api.NewClient(&api.Config{Address: vaultAddr})
	if err != nil {
		logrus.Errorf("wrapData: Error creating Vault client: %v", err)
		return "", nil, fmt.Errorf("wrapData: failed to create Vault client: %w",
			err)
	}

	client.SetToken(vaultToken)

	// Convert TTL to time.Duration
	ttlDuration, err := time.ParseDuration(ttl + "s")
	if err != nil {
		logrus.Errorf("wrapData: Invalid TTL: %v, TTL: %s", err, ttl)
		return "", nil, fmt.Errorf("wrapData: invalid TTL: %w, TTL: %s", err, ttl)
	}
	ttlString := ttlDuration.String()

	// Prepare the data to wrap
	requestData := map[string]interface{}{
		"data": data,
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a logical request
	req := client.NewRequest("POST", "/v1/sys/wrapping/wrap")

	// Set the Wrap-TTL header
	req.Headers.Set("X-Vault-Wrap-TTL", ttlString)

	// Set the request body
	if err := req.SetJSONBody(requestData); err != nil {
		logrus.Errorf("wrapData: Error setting request body: %v", err)
		return "", nil, fmt.Errorf("wrapData: failed to set request body: %w",
			err)
	}

	// Send the request with context
	resp, err := client.RawRequestWithContext(ctx, req)
	if err != nil {
		logrus.Errorf("wrapData: Error wrapping data: %v", err)
		return "", nil, fmt.Errorf("wrapData: failed to wrap data: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		logrus.Errorf("wrapData: Error parsing wrap response: %v", err)
		return "", nil, fmt.Errorf("wrapData: failed to parse wrap response: %w",
			err)
	}

	return secret.WrapInfo.Token, secret.WrapInfo, nil
}

func unwrapData(token string) (map[string]interface{}, error) {
	client, err := api.NewClient(&api.Config{Address: vaultAddr})
	if err != nil {
		logrus.Errorf("unwrapData: Error creating Vault client: %v", err)
		return nil, fmt.Errorf("unwrapData: failed to create Vault client: %w",
			err)
	}

	client.SetToken(token)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Unwrap the data with context
	secret, err := client.Logical().UnwrapWithContext(ctx, "")
	if err != nil {
		logrus.Errorf("unwrapData: Error unwrapping data: %v, Token: %s", err,
			token)
		return nil, fmt.Errorf("unwrapData: failed to unwrap data: %w, Token: %s",
			err, token)
	}

	return secret.Data, nil
}
