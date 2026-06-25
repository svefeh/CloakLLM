# CloakLLM

[![Build & Push Image](https://github.com/svefeh/cloakllm/actions/workflows/build-image.yml/badge.svg)](https://github.com/svefeh/cloakllm/actions/workflows/build-image.yml)
[![Lint and Quality Checks](https://github.com/svefeh/cloakllm/actions/workflows/lint.yml/badge.svg)](https://github.com/svefeh/cloakllm/actions/workflows/lint.yml)
[![Dependabot Enabled](https://img.shields.io/badge/Dependabot-enabled-0366d6?logo=dependabot)](https://github.com/svefeh/cloakllm/network/updates)

**CloakLLM** is a stateless, end-to-end encrypted (E2EE) API routing bridge. It allows developers to safely tunnel sensitive JSON payloads (such as LLM prompts and API tokens) through untrusted intermediate networks using robust AES-256-GCM encryption. 

By encrypting the entire routing package including the target endpoint, API token, and payload CloakLLM ensures that intermediary nodes, network proxies, or deep packet inspection (DPI) appliances cannot read or manipulate your data. It acts as a drop-in, locally hosted OpenAI-compatible endpoint for UI frontends like Open WebUI.

---

## 🏗 Architecture

CloakLLM splits the routing process into two tightly coupled, stateless components to minimize the attack surface:

1. **The Local Daemon (Golang):** Runs locally on your machine or within a container pod. It intercepts standard HTTP requests, wraps the payload and authentication tokens into a secure JSON package, encrypts it using a pre-shared 256-bit key, and forwards it to your remote proxy.
2. **The Server Proxy (Python CGI):** A lightweight script hosted on any standard web server. It receives the encrypted blob, decrypts it in memory, extracts the target URL and token, forwards the request to the upstream LLM API (OpenAI, Gemini, etc.), and securely encrypts the response for the journey back.

### Traffic Flow
```text
[ Open WebUI ] 
      │ (Plaintext JSON to localhost:8080)
      ▼
[ CloakLLM Local Daemon ] 
      │ (AES-GCM Encrypted Payload via HTTPS POST)
      ▼
[ Intermediate Network / DPI Firewall ]  <-- Traffic appears as opaque binary/hex data
      │
      ▼
[ CloakLLM Server Proxy (endpoint.cgi) ] 
      │ (Decrypts and forwards via HTTPS)
      ▼
[ Target API (e.g., Google Gemini / OpenAI) ]

```

---

## ✨ Key Features

* **True End-to-End Encryption:** Utilizes standard `AES-256-GCM` for authenticated encryption. Both the payload and the routing metadata are secured against interception and manipulation.
* **Zero-State Design:** The remote server proxy stores no state, caches no API keys, and hardcodes no target URLs.
* **Drop-in Compatibility:** The daemon natively mimics the OpenAI API structure (`/v1/chat/completions` and `/v1/models`), making it plug-and-play with almost any modern AI frontend.
* **OCI Container Ready:** Distributed as a minimal, highly optimized container image via the GitHub Container Registry.
* **Infrastructure Agnostic:** The Python proxy is designed to run in highly restricted shared-hosting environments using pure CGI, requiring no persistent background processes.

---

## 🛡️ Security & Verification

This project embraces a Zero-Trust supply chain. All container images pushed to the GitHub Container Registry (GHCR) are **cryptographically signed** using Sigstore/Cosign via Keyless OIDC authentication.

You can verify the immutable integrity and origin of the container image before running it by using the `cosign` CLI:

```bash
cosign verify \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  --certificate-identity="https://github.com/svefeh/cloakllm/.github/workflows/build-image.yml@refs/heads/main" \
  ghcr.io/svefeh/cloakllm:latest

```

---

## 🚀 Deployment Guide

### Prerequisites
Before deploying, ensure your environment meets these requirements:
* **Local Machine / Pod:** [Go 1.22+](https://go.dev/dl/) (if building from source) and [Podman](https://podman.io/) (for running the container).
* **Remote Server:** A web server supporting **CGI scripts** (e.g., Apache).
* **Dependencies (Server):** The Python environment on the remote server must have the `cryptography` library installed. 
  * *Tip:* If you don't have root access on the server, you can install the library locally in your home directory using `pip install --target=$HOME/python_libs cryptography` and prepend this path in the shebang line of `endpoint.cgi`.
* **Network:** Outbound HTTPS connectivity from your server proxy to the upstream LLM provider (e.g., Google's APIs).

### 1. Remote Server Proxy (Python CGI)
1. Open `server/endpoint.cgi` and replace the `SHARED_KEY` variable with a secure, random 64-character hex string.
   * *Tip: You can generate one via terminal using `openssl rand -hex 32`.*
2. Rename `endpoint.cgi` to blend into your hosting environment (e.g., `api_health.cgi`).
3. Upload the script to your web server's `cgi-bin` directory via FTP (**using ASCII mode** to preserve Unix line endings).
4. Set the file permissions to executable (`chmod 755`).

### 2. Local Daemon (Podman / GHCR)
The most robust way to run the local daemon is via the official container image hosted on GitHub Container Registry (GHCR).

```bash
# Set your environment variables (Configured for Google Gemini API)
export CLOAK_SHARED_KEY="YOUR_64_CHARACTER_HEX_KEY"
export CLOAK_PROXY_URL="[https://your-domain.com/cgi-bin/api_health.cgi](https://your-domain.com/cgi-bin/api_health.cgi)"

# Using Google's official OpenAI-compatible endpoints for Gemini
export CLOAK_TARGET_API="[https://generativelanguage.googleapis.com/v1beta/openai/chat/completions](https://generativelanguage.googleapis.com/v1beta/openai/chat/completions)"
export CLOAK_MODELS_API="[https://generativelanguage.googleapis.com/v1beta/openai/models](https://generativelanguage.googleapis.com/v1beta/openai/models)"

# Run the container using Podman
podman run -d \
  --name cloakllm \
  -p 8080:8080 \
  -e CLOAK_SHARED_KEY=$CLOAK_SHARED_KEY \
  -e CLOAK_PROXY_URL=$CLOAK_PROXY_URL \
  -e CLOAK_TARGET_API=$CLOAK_TARGET_API \
  -e CLOAK_MODELS_API=$CLOAK_MODELS_API \
  ghcr.io/svefeh/cloakllm:latest
```

### 3. Integration with Open WebUI

1. Navigate to **Settings > Connections** in Open WebUI.
2. Add a new OpenAI-compatible connection.
3. Set the **API Base URL** to `http://localhost:8080/v1`.
4. Set the **API Key** to your actual target API key (e.g., your real Gemini token).
* *Note: Open WebUI will send this token to CloakLLM, which will encrypt it and tunnel it to the server proxy.*



---

## 🛠 Local Development & Contributing

To maintain a high standard of code quality and a readable repository history, this project enforces **Conventional Commits** and automated linting via `pre-commit`.

### 1. Environment Setup
Before contributing, ensure you have the required tools installed. We recommend using `pipx` to install Python-based CLI tools in isolated environments:

```bash
# Install pre-commit and commitizen globally
pipx install pre-commit
pipx install commitizen

# Clone the repository
git clone [https://github.com/svefeh/cloakllm.git](https://github.com/svefeh/cloakllm.git)
cd cloakllm

# Initialize the Git hooks
make init

```

### 2. Making Commits

Do not use the standard `git commit` command. Instead, use the interactive Commitizen CLI. This guarantees that your commit messages comply with the standard and automatically triggers the pre-commit linters (e.g., `golangci-lint`, whitespace checks, and JSON validation) before saving your changes.

```bash
# Stage your changes
git add .

# Create a commit interactively
cz c
```

### 3. Building & Testing

If you prefer to compile the daemon from source or build the OCI container locally:

```bash
# Clone the repository
git clone [https://github.com/svefeh/cloakllm.git](https://github.com/svefeh/cloakllm.git)
cd cloakllm

# Build the binary locally for Linux
make build-linux

# Or build the OCI image locally using Podman
make container

# Run tests
make test

```

---

## 📜 License & Disclaimer

This project is licensed under the **Apache License 2.0** - see the `LICENSE` file for details.

**Disclaimer:** This software is provided "as is", without warranty of any kind. It is intended as an educational proof-of-concept for stateless, E2EE routing. Ensure you have the proper authorization before tunneling traffic across network boundaries.
