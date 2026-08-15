package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
)

const (
	vaultEnvelopeFormat = "vault-wrapper/base64-chunks-v1"
	// Encoded chunks are 256 KiB, safely below Vault's default 1 MiB
	// max_json_string_value_length while keeping the envelope compact.
	vaultPayloadChunkSize = 192 * 1024
	vaultOperationTimeout = 30 * time.Second
)

type vaultPayloadEnvelope struct {
	Format    string   `json:"format"`
	Chunks    []string `json:"chunks"`
	ReceiptID string   `json:"receipt_id,omitempty"`
}

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
		log.Println("WARN  VAULT_TOKEN not set, using insecure development default: root")
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

func wrapData(data string, ttl string, receiptID string) (string, *api.SecretWrapInfo, error) {
	// Validate TTL is a parseable duration.
	if _, err := time.ParseDuration(ttl + "s"); err != nil {
		return "", nil, fmt.Errorf("wrapData: invalid TTL %q: %w", ttl, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), vaultOperationTimeout)
	defer cancel()

	req := vaultClient.NewRequest("POST", "/v1/sys/wrapping/wrap")
	req.WrapTTL = ttl + "s"
	if err := req.SetJSONBody(map[string]interface{}{"data": encodeVaultPayload(data, receiptID)}); err != nil {
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
	if secret == nil || secret.WrapInfo == nil || secret.WrapInfo.Token == "" {
		return "", nil, fmt.Errorf("wrapData: Vault response did not include wrapping information")
	}

	return secret.WrapInfo.Token, secret.WrapInfo, nil
}

func encodeVaultPayload(data string, receiptID string) vaultPayloadEnvelope {
	chunks := make([]string, 0, (len(data)+vaultPayloadChunkSize-1)/vaultPayloadChunkSize)
	for start := 0; start < len(data); start += vaultPayloadChunkSize {
		end := min(start+vaultPayloadChunkSize, len(data))
		chunks = append(chunks, base64.StdEncoding.EncodeToString([]byte(data[start:end])))
	}
	if len(chunks) == 0 {
		chunks = append(chunks, "")
	}
	return vaultPayloadEnvelope{Format: vaultEnvelopeFormat, Chunks: chunks, ReceiptID: receiptID}
}

// decodeVaultPayload accepts the chunked envelope as well as the legacy plain
// string so links created by older versions remain usable.
func decodeVaultPayload(value interface{}) (string, string, error) {
	if legacy, ok := value.(string); ok {
		return legacy, "", nil
	}

	envelope, ok := value.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("unexpected payload type %T", value)
	}
	format, ok := envelope["format"].(string)
	if !ok || format != vaultEnvelopeFormat {
		return "", "", fmt.Errorf("unsupported payload envelope")
	}
	rawChunks, ok := envelope["chunks"].([]interface{})
	if !ok || len(rawChunks) == 0 {
		return "", "", fmt.Errorf("payload envelope has no chunks")
	}
	receiptID, _ := envelope["receipt_id"].(string)
	if receiptID != "" && !validOpaqueID(receiptID) {
		return "", "", fmt.Errorf("payload envelope has an invalid receipt")
	}

	var decoded strings.Builder
	for i, rawChunk := range rawChunks {
		chunk, ok := rawChunk.(string)
		if !ok {
			return "", "", fmt.Errorf("payload chunk %d is not a string", i)
		}
		part, err := base64.StdEncoding.DecodeString(chunk)
		if err != nil {
			return "", "", fmt.Errorf("decode payload chunk %d: %w", i, err)
		}
		if int64(decoded.Len()+len(part)) > requestBodyLimit() {
			return "", "", fmt.Errorf("decoded payload exceeds the request limit")
		}
		decoded.Write(part)
	}
	return decoded.String(), receiptID, nil
}

func unwrapData(token string) (map[string]interface{}, error) {
	// Clone the shared client so we can set the wrapping token without
	// causing a data race on the shared client's token field.
	c, err := vaultClient.Clone()
	if err != nil {
		return nil, fmt.Errorf("unwrapData: failed to clone client: %w", err)
	}
	c.SetToken(token)

	ctx, cancel := context.WithTimeout(context.Background(), vaultOperationTimeout)
	defer cancel()

	secret, err := c.Logical().UnwrapWithContext(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("unwrapData: failed to unwrap: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("unwrapData: Vault returned an empty response")
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

	if secret == nil {
		return nil, fmt.Errorf("lookupWrappingToken: Vault returned an empty response")
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	h, err := vaultClient.Sys().HealthWithContext(ctx)
	if err != nil {
		log.Printf("getVaultHealth: error from Vault: %v", err)
		return VaultHealth{Status: "unhealthy", Message: "Vault health check failed"}
	}
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
