# Vault Data Wrapper

Vault Data Wrapper is a web application that leverages HashiCorp Vault's wrapping feature to securely store and share temporary data, including text snippets, code, and files. It generates time-limited tokens for secure data exchange.

## Features

*   **🔒 Secure Data Wrapping/Unwrapping:** Wraps data with a specified Time-To-Live (TTL), generating a unique token.
*   **🔗 Shareable URLs:** Creates shareable URLs containing the wrapped token for easy access.
*   **✨ User-Friendly Interface:** Provides a modern, responsive UI with syntax highlighting (CodeMirror).
*   **🎨 Persistent Theme:** Remembers user's preferred theme (light/dark) across sessions.
*   **📋 Clipboard Integration:** Simplifies copying tokens and URLs.
*   **📁 File Upload Support:** Allows wrapping of files up to 5MB.

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

*   **Vault Address:** `http://vault:8200` (configurable via `VAULT_ADDR` environment variable)
*   **Vault Token:** `root` (configurable via `VAULT_TOKEN` environment variable - **Use a secure token in production!**)

### Usage

1.  **Wrap Data:** Enter text/code or upload files, set TTL, and click "Wrap". Copy the generated token or shareable link.
2.  **Unwrap Data:** Open the shareable link or paste the token into the "Unwrap" field and click "Unwrap".

## Screenshots

![alt text](image.png)

## Built With

*   [Go](https://golang.org/)
*   [HashiCorp Vault](https://www.vaultproject.io/)
*   [Docker](https://www.docker.com/)
*   [CodeMirror](https://codemirror.net/)
