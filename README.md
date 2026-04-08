# URL Shortener

A lightweight, high-performance URL shortener written in Go. The service generates short, alphanumeric codes for long URLs and redirects visitors to the original destination.

---

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [How It Works](#how-it-works)
- [API Reference](#api-reference)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Running the Server](#running-the-server)
- [Usage Examples](#usage-examples)
- [Configuration](#configuration)
- [Dependencies](#dependencies)
- [License](#license)

---

## Features

- **URL Shortening** – Submit any long URL and receive a short, unique code.
- **Instant Redirection** – Visiting a short URL performs an HTTP 301 redirect to the original URL.
- **Base-62 Encoding** – Short codes are generated using a base-62 alphabet (`a-z`, `A-Z`, `0-9`), keeping URLs compact and human-readable.
- **Thread-Safe In-Memory Store** – All URL mappings are stored in memory using a `sync.RWMutex`-protected map, making it safe for concurrent requests.
- **Extensible Storage Interface** – The `URLStore` interface makes it straightforward to swap the in-memory store for a persistent backend (e.g., PostgreSQL or Redis).
- **Atomic Counter** – A global atomic counter guarantees unique IDs across concurrent shortening requests without locks.
- **Zero External Runtime Dependencies** – Uses only the Go standard library at runtime; no database or cache is required to run the server.

---

## Project Structure

```
Url-Shortner/
├── cmd/
│   └── api/
│       └── main.go          # HTTP server, route handlers, entry point
├── internal/
│   ├── shortner/
│   │   └── shortner.go      # Base-62 encoding logic
│   └── store/
│       └── store.go         # URLStore interface + in-memory implementation
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## How It Works

1. **Shorten a URL** – A `POST /shorten` request is made with a JSON body containing the long URL.
2. **ID Generation** – An atomic counter (starting at `10001`) is incremented for each request, ensuring every URL gets a unique numeric ID.
3. **Base-62 Encoding** – The numeric ID is encoded into a compact alphanumeric string using the 62-character alphabet `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`.
4. **Storage** – The short code and the original URL are stored in a thread-safe in-memory map.
5. **Redirect** – A `GET /{code}` request looks up the code in the store and issues an HTTP `301 Moved Permanently` redirect to the original URL.

---

## API Reference

### `POST /shorten`

Shortens a given URL.

**Request Body**

```json
{
  "url": "https://example.com/some/very/long/path"
}
```

**Response** `200 OK`

```json
{
  "short_url": "http://localhost:8080/3o"
}
```

**Error Responses**

| Status | Reason |
|--------|--------|
| `400 Bad Request` | Request body is missing or cannot be parsed as JSON |

---

### `GET /{code}`

Redirects the caller to the original URL associated with `code`.

**Response** `301 Moved Permanently` — `Location` header contains the original URL.

**Error Responses**

| Status | Reason |
|--------|--------|
| `404 Not Found` | No URL is registered for the given code |

---

## Getting Started

### Prerequisites

- [Go 1.21+](https://golang.org/dl/) (the module uses Go 1.25; any recent toolchain works)

### Installation

```bash
git clone https://github.com/fierceadventurer/Url-Shortner.git
cd Url-Shortner
go mod download
```

### Running the Server

```bash
go run ./cmd/api
```

The server starts on **port 8080**:

```
Server listening on :8080
```

---

## Usage Examples

**Shorten a URL**

```bash
curl -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://www.github.com/fierceadventurer/Url-Shortner"}'
```

Response:

```json
{
  "short_url": "http://localhost:8080/3o"
}
```

**Follow the short URL (browser or curl)**

```bash
curl -L http://localhost:8080/3o
```

This resolves the code and redirects to the original URL.

---

## Configuration

The application reads environment variables from a `.env` file (loaded via `godotenv`) when present. A `.env` file is git-ignored by default.

| Variable | Description | Default |
|----------|-------------|---------|
| *(none required for in-memory mode)* | | |

> **Note:** The `go.mod` file already includes driver packages for **PostgreSQL** (`lib/pq`) and **Redis** (`go-redis/v9`). These are available to plug in by implementing the `URLStore` interface with a persistent backend.

---

## Dependencies

| Package | Purpose |
|---------|---------|
| [`github.com/joho/godotenv`](https://github.com/joho/godotenv) | Load environment variables from a `.env` file |
| [`github.com/lib/pq`](https://github.com/lib/pq) | PostgreSQL driver (for future persistent storage) |
| [`github.com/redis/go-redis/v9`](https://github.com/redis/go-redis) | Redis client (for future caching / persistent storage) |
| [`go.uber.org/atomic`](https://github.com/uber-go/atomic) | Strongly-typed atomic primitives |

---

## License

This project is licensed under the [MIT License](LICENSE).  
Copyright © 2026 Ayush Tripathi.
