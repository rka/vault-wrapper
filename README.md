# Vault Data Wrapper

Vault Data Wrapper is a web application that leverages HashiCorp Vault's wrapping feature to securely store and share temporary data, including text snippets, code, and files. It generates time-limited tokens for secure data exchange.

## Features

*   **🔒 Secure Data Wrapping/Unwrapping:** Wraps data with a specified Time-To-Live (TTL), generating a unique token.
*   **🔗 Safer Shareable URLs:** Keeps wrapped tokens in the URL fragment so they are not sent in HTTP requests or referrer headers.
*   **✨ Responsive Interface:** Provides a focused create/open workflow with syntax highlighting, keyboard-accessible controls, and light/dark themes.
*   **🎨 Persistent Theme:** Remembers user's preferred theme (light/dark) across sessions.
*   **📋 Clipboard Integration:** Simplifies copying tokens and URLs.
*   **📁 File Upload Support:** Allows a combined decoded payload of up to 5MB by default.

## Getting Started

### Prerequisites

*   [Docker](https://www.docker.com/) and [Docker Compose](https://docs.docker.com/compose/)

### Installation

1.  **Clone the Repository:**

    ```bash
    git clone <repository_url>
    cd vault-data-wrapper
    ```

2.  **Run with Docker Compose:**

    ```bash
    docker-compose up -d --build
    ```

3.  **Access the Application:**

    Open your web browser and navigate to:

    ```
    http://localhost:3001
    ```

    (Adjust `docker-compose.yml` for custom configurations.)

### Configuration

> [!WARNING]
> The included Compose file starts Vault in in-memory development mode with a root token and TLS disabled. It is suitable for local testing only; use an initialized, persistent Vault deployment and a least-privilege token in production.

*   **Vault Address:** `http://vault:8200` (configurable via `VAULT_ADDR` environment variable)
*   **Vault Token:** `root` (configurable via `VAULT_TOKEN` environment variable - **Use a secure token in production!**)
*   **Payload Limit:** `5242880` bytes (configurable via `MAX_REQUEST_SIZE`)
*   **Proxy Headers:** Set `TRUST_PROXY_HEADERS=true` only when the app is reachable exclusively through a trusted reverse proxy.

### Usage

1.  **Wrap Data:** Enter text/code or upload files, set TTL, and click "Wrap". Copy the generated token or shareable link.
2.  **Unwrap Data:** Open the shareable link or paste the token, review the one-time-use warning, and explicitly confirm retrieval. Loading a link does not consume its token.

## CLI Usage

You can interact with the API directly using command-line tools.

### curl (Linux, macOS, Windows Git Bash)

**Wrap Data:**
```bash
curl -X POST http://localhost:3001/wrap \
  -H "Content-Type: application/json" \
  -d '{"data": {"text": "My Secret"}, "ttl": "3600"}'
```

**Unwrap Data:**
```bash
curl -X POST http://localhost:3001/unwrap \
  -H "Content-Type: application/json" \
  -d '{"token": "YOUR_TOKEN"}'
```

### PowerShell (Windows)

**Wrap Data:**
```powershell
Invoke-RestMethod -Uri "http://localhost:3001/wrap" -Method Post -ContentType "application/json" -Body (@{data=@{text="My Secret"}; ttl="3600"} | ConvertTo-Json) | ConvertTo-Json
```

**Unwrap Data:**
```powershell
Invoke-RestMethod -Uri "http://localhost:3001/unwrap" -Method Post -ContentType "application/json" -Body (@{token="YOUR_TOKEN"} | ConvertTo-Json) | ConvertTo-Json
```

## Development checks

```bash
go build ./...
go vet ./...
node --check static/app.js
```

The API accepts JSON requests only, enforces a 30-day maximum wrapping TTL, validates decoded attachment sizes, and chunks its internal Vault envelope so permitted payloads stay below Vault's per-string JSON limit.

## Built With

*   [Go](https://golang.org/)
*   [HashiCorp Vault](https://www.vaultproject.io/)
*   [Docker](https://www.docker.com/)
*   [CodeMirror](https://codemirror.net/)
