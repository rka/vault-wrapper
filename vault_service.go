package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
)

var (
	vaultAddr  = os.Getenv("VAULT_ADDR")  // Get Vault address from environment
	vaultToken = os.Getenv("VAULT_TOKEN") // Get Vault token from environment
)

func init() {
	if vaultAddr == "" {
		vaultAddr = "http://vault:8200" // Default value if not set
		log.Println("VAULT_ADDR not set, using default value: http://vault:8200")
	}
	if vaultToken == "" {
		vaultToken = "root" // Default value if not set
		log.Println("VAULT_TOKEN not set, using default value: root")
	}
}

func wrapData(data string, ttl string) (string, *api.SecretWrapInfo, error) {
	client, err := api.NewClient(&api.Config{Address: vaultAddr})
	if (err != nil) {
		log.Printf("wrapData: Error creating Vault client: %v\n", err)
		return "", nil, fmt.Errorf("wrapData: failed to create Vault client: %w", err)
	}

	client.SetToken(vaultToken)

	// Convert TTL to time.Duration
	_, err = time.ParseDuration(ttl + "s")
	if err != nil {
		log.Printf("wrapData: Invalid TTL: %v, TTL: %s\n", err, ttl)
		return "", nil, fmt.Errorf("wrapData: invalid TTL: %w, TTL: %s", err, ttl)
	}
	// Use the input TTL directly with 's' suffix to ensure it's treated as seconds
	ttlString := ttl + "s"

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
		log.Printf("wrapData: Error setting request body: %v\n", err)
		return "", nil, fmt.Errorf("wrapData: failed to set request body: %w", err)
	}

	// Send the request with context
	resp, err := client.RawRequestWithContext(ctx, req)
	if err != nil {
		log.Printf("wrapData: Error wrapping data: %v\n", err)
		return "", nil, fmt.Errorf("wrapData: failed to wrap data: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		log.Printf("wrapData: Error parsing wrap response: %v\n", err)
		return "", nil, fmt.Errorf("wrapData: failed to parse wrap response: %w", err)
	}

	return secret.WrapInfo.Token, secret.WrapInfo, nil
}

func unwrapData(token string) (map[string]interface{}, error) {
	client, err := api.NewClient(&api.Config{Address: vaultAddr})
	if err != nil {
		log.Printf("unwrapData: Error creating Vault client: %v\n", err)
		return nil, fmt.Errorf("unwrapData: failed to create Vault client: %w", err)
	}

	client.SetToken(token)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Unwrap the data with context
	secret, err := client.Logical().UnwrapWithContext(ctx, "")
	if err != nil {
		log.Printf("unwrapData: Error unwrapping data: %v, Token: %s\n", err, token)
		return nil, fmt.Errorf("unwrapData: failed to unwrap data: %w, Token: %s", err, token)
	}

	return secret.Data, nil
}

func lookupWrappingToken(token string) (*api.Secret, error) {
	client, err := api.NewClient(&api.Config{Address: vaultAddr})
	if err != nil {
		log.Printf("lookupWrappingToken: Error creating Vault client: %v\n", err)
		return nil, fmt.Errorf("lookupWrappingToken: failed to create Vault client: %w", err)
	}

	// For wrapping token lookup, we don't need to set the token
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a logical request for wrapping token lookup
	req := client.NewRequest("POST", "/v1/sys/wrapping/lookup")
	if err := req.SetJSONBody(map[string]interface{}{
		"token": token,
	}); err != nil {
		return nil, fmt.Errorf("lookupWrappingToken: failed to set request body: %w", err)
	}

	// Send the request
	resp, err := client.RawRequestWithContext(ctx, req)
	if err != nil {
		log.Printf("lookupWrappingToken: Error looking up token: %v, Token: %s\n", err, token)
		return nil, fmt.Errorf("lookupWrappingToken: failed to lookup token: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lookupWrappingToken: failed to parse response: %w", err)
	}

	return secret, nil
}
