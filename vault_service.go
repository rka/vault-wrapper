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

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Convert TTL to time.Duration
    ttlDuration, err := time.ParseDuration(ttl + "s")
    if err != nil {
        logrus.Error("Invalid TTL: ", err)
        return "", nil, err
    }

    secret, err := client.Logical().WriteWithContext(ctx, "sys/wrapping/wrap", map[string]interface{}{
        "ttl":  ttlDuration.String(),
        "body": data,
    })
    if err != nil {
        logrus.Error("Error wrapping data: ", err)
        return "", nil, err
    }

    return secret.WrapInfo.Token, secret.WrapInfo, nil
}

func unwrapData(token string) (string, error) {
    client, err := api.NewClient(&api.Config{Address: vaultAddr})
    if err != nil {
        logrus.Error("Error creating Vault client: ", err)
        return "", err
    }

    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    client.SetToken(token)

    secret, err := client.Logical().UnwrapWithContext(ctx, "")
    if err != nil {
        logrus.Error("Error unwrapping data: ", err)
        return "", err
    }

    data, ok := secret.Data["body"].(string)
    if !ok {
        logrus.Error("Error retrieving unwrapped data")
        return "", err
    }

    return data, nil
}
