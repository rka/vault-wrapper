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
	vaultAddr   = os.Getenv("VAULT_ADDR")
	vaultToken  = os.Getenv("VAULT_TOKEN")
	vaultClient *api.Client
)

func init() {
	if vaultAddr == "" {
		vaultAddr = "http://vault:8200"
		log.Println("VAULT_ADDR not set, using default: http://vault:8200")
	}
	if vaultToken == "" {
		vaultToken = "root"
		log.Println("VAULT_TOKEN not set, using default: root")
	}
}

// initVaultClient creates the single shared Vault API client used by all service functions.
// It must be called once before any service function is invoked.
func initVaultClient() error {
	cfg := api.DefaultConfig()
	cfg.Address = vaultAddr

	c, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("initVaultClient: failed to create client: %w", err)
	}
	c.SetToken(vaultToken)
	vaultClient = c
	log.Printf("Vault client initialized (addr=%s)", vaultAddr)
	return nil
}

func wrapData(data string, ttl string) (string, *api.SecretWrapInfo, error) {
	// Validate TTL is a parseable duration.
	if _, err := time.ParseDuration(ttl + "s"); err != nil {
		return "", nil, fmt.Errorf("wrapData: invalid TTL %q: %w", ttl, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := vaultClient.NewRequest("POST", "/v1/sys/wrapping/wrap")
	req.WrapTTL = ttl + "s"
	if err := req.SetJSONBody(map[string]interface{}{"data": data}); err != nil {
		return "", nil, fmt.Errorf("wrapData: failed to set request body: %w", err)
	}

	resp, err := vaultClient.RawRequestWithContext(ctx, req)
	if err != nil {
		return "", nil, fmt.Errorf("wrapData: Vault request failed: %w", err)
	}
	defer resp.Body.Close()

	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("wrapData: failed to parse response: %w", err)
	}

	return secret.WrapInfo.Token, secret.WrapInfo, nil
}

func unwrapData(token string) (map[string]interface{}, error) {
	// Clone the shared client so we can set the wrapping token without
	// causing a data race on the shared client's token field.
	c, err := vaultClient.Clone()
	if err != nil {
		return nil, fmt.Errorf("unwrapData: failed to clone client: %w", err)
	}
	c.SetToken(token)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	secret, err := c.Logical().UnwrapWithContext(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("unwrapData: failed to unwrap: %w", err)
	}

	return secret.Data, nil
}

func lookupWrappingToken(token string) (*api.Secret, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := vaultClient.NewRequest("POST", "/v1/sys/wrapping/lookup")
	if err := req.SetJSONBody(map[string]interface{}{"token": token}); err != nil {
		return nil, fmt.Errorf("lookupWrappingToken: failed to set body: %w", err)
	}

	resp, err := vaultClient.RawRequestWithContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("lookupWrappingToken: request failed: %w", err)
	}
	defer resp.Body.Close()

	secret, err := api.ParseSecret(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lookupWrappingToken: failed to parse response: %w", err)
	}

	return secret, nil
}

// VaultHealth describes the current state of the Vault backend.
type VaultHealth struct {
	Status      string `json:"status"` // "healthy" | "standby" | "unhealthy"
	Initialized bool   `json:"initialized"`
	Sealed      bool   `json:"sealed"`
	Standby     bool   `json:"standby"`
	Version     string `json:"vault_version,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	Message     string `json:"message,omitempty"`
}

// getVaultHealth polls Vault's health endpoint and returns a structured status.
// A 3-second context guards against a hung Vault instance.
func getVaultHealth() VaultHealth {
	type result struct {
		h   *api.HealthResponse
		err error
	}

	ch := make(chan result, 1)
	go func() {
		h, err := vaultClient.Sys().Health()
		ch <- result{h, err}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		log.Println("getVaultHealth: health check timed out")
		return VaultHealth{Status: "unhealthy", Message: "health check timed out"}
	case r := <-ch:
		if r.err != nil {
			log.Printf("getVaultHealth: error from Vault: %v", r.err)
			return VaultHealth{Status: "unhealthy", Message: r.err.Error()}
		}
		h := r.h
		status := "healthy"
		if !h.Initialized || h.Sealed {
			status = "unhealthy"
		} else if h.Standby {
			status = "standby"
		}
		return VaultHealth{
			Status:      status,
			Initialized: h.Initialized,
			Sealed:      h.Sealed,
			Standby:     h.Standby,
			Version:     h.Version,
			ClusterName: h.ClusterName,
		}
	}
}
