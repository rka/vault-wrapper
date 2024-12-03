# Vault Data Wrapper POC

Vault Data Wrapper is a web application that utilizes Hashicorp Vault's wrap feature to securely store and share temporary text and code snippets. It allows users to wrap data into a token with a specified Time-To-Live (TTL) and generate a shareable URL. Recipients can use this URL to unwrap and view the original data securely.

## Features

- **Secure Data Wrapping and Unwrapping**
  - Wrap any text/code snippet or files with a specified TTL.
  - Generate a unique wrapped token for secure sharing.
- **Shareable Links**
  - Create shareable URLs containing the wrapped token.
  - Auto-populate and unwrap data when accessing the shared link.
- **Modern User Interface**
  - Sleek and responsive design with light and dark modes.
  - Syntax-highlighted code editing using CodeMirror.
- **Persistent Dark Mode**
  - Users' theme preference is saved and persists across sessions.
- **Copy-to-Clipboard Functionality**
  - Easily copy wrapped tokens and shareable links with a single click.

## Screenshots

![image](https://github.com/user-attachments/assets/659d14de-87a7-40c1-be52-6f0644211026)


## Getting Started

### Prerequisites

- **Docker** and **Docker Compose** installed on your system.

### Installation

1. **Clone the Repository**

   ```bash
   git clone https://github.com/yourusername/vault-data-wrapper.git
   cd vault-data-wrapper
   ```

1. **Build and Run the Application**

   ```bash
   docker-compose up -d --build
   ```

1. **Access the Application**

   Open your web browser and navigate to:

   ```
   http://localhost:3001
   ```

   > 💡update `docker-compose.yml` as needed

### Configuration

- The application is currently configured to use a development instance of Hashicorp Vault.
- Default Vault address: `http://vault:8200`
- Default Vault token: `root` (For development purposes only. Do not use in production.)

## Usage

### Wrapping Data

1. **Enter Data**

   - Input the text or code you wish to wrap in the editor provided.

2. **Set TTL**

   - Specify the Time-To-Live (TTL) in seconds for how long the wrapped token should be valid.

3. **Wrap Data**

   - Click the **Wrap** button.
   - The application will generate a wrapped token and a shareable link.
   - Click on the token or link fields to copy them to your clipboard.

4. **Share**

   - Share the wrapped token or the generated URL with the intended recipient.

### Unwrapping Data

1. **Using the Shareable Link**

   - Open the shared URL in a web browser.
   - The application will automatically populate the token field and unwrap the data.

2. **Using the Wrapped Token**

   - Navigate to the application.
   - Paste the wrapped token into the **Unwrap** field.
   - Click the **Unwrap** button to view the original data.



## Built With

- [Go](https://golang.org/) - Backend language.
- [Hashicorp Vault](https://www.vaultproject.io/) - Secure secret storage.
- [Docker](https://www.docker.com/) - Containerization platform.
- [CodeMirror](https://codemirror.net/) - In-browser code editor.
- [Logrus](https://github.com/sirupsen/logrus) - Logging library.

