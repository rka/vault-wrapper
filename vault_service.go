package main

import (
    "context"
    "time"

    "github.com/hashicorp/vault/api"
    "github.com/sirupsen/logrus"
)

const (
    vaultAddr  = "http://vault:8200" // Update as per your Vault address
    vaultToken = "root"              // Use a secure token in production
)

func wrapData(data string, ttl string) (string, *api.SecretWrapInfo, error) {
    client, err := api.NewClient(&api.Config{Address: vaultAddr})
    if err != nil {
        logrus.Error("Error creating Vault client: ", err)
        return "", nil, err
    }

    client.SetToken(vaultToken)

    // Convert TTL to time.Duration
    ttlDuration, err := time.ParseDuration(ttl + "s")
    if err != nil {
        logrus.Error("Invalid TTL: ", err)
        return "", nil, err
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
        logrus.Error("Error setting request body: ", err)
        return "", nil, err
    }

    // Send the request with context
    resp, err := client.RawRequestWithContext(ctx, req)
    if err != nil {
        logrus.Error("Error wrapping data: ", err)
        return "", nil, err
    }
    defer resp.Body.Close()

    // Parse the response
    secret, err := api.ParseSecret(resp.Body)
    if err != nil {
        logrus.Error("Error parsing wrap response: ", err)
        return "", nil, err
    }

    return secret.WrapInfo.Token, secret.WrapInfo, nil
}

func unwrapData(token string) (map[string]interface{}, error) {
    client, err := api.NewClient(&api.Config{Address: vaultAddr})
    if err != nil {
        logrus.Error("Error creating Vault client: ", err)
        return nil, err
    }

    client.SetToken(token)

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Unwrap the data with context
    secret, err := client.Logical().UnwrapWithContext(ctx, "")
    if err != nil {
        logrus.Error("Error unwrapping data: ", err)
        return nil, err
    }

    return secret.Data, nil
}
