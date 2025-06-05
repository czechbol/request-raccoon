# 🦝 Request Raccoon

<p align="center">
  <a href="https://golang.org">
    <img src="https://img.shields.io/badge/Go-1.24+-blue?style=for-the-badge" alt="Go Version">
  </a>
  <a href="https://github.com/czechbol/request-raccoon/actions">
    <img src="https://img.shields.io/github/actions/workflow/status/czechbol/request-raccoon/ci.yml?branch=main&style=for-the-badge" alt="Build Status">
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge" alt="License: MIT">
  </a>
  <a href="https://hub.docker.com/r/czechbol/request-raccoon">
    <img src="https://img.shields.io/badge/Docker-Ready-blue?style=for-the-badge" alt="Docker">
  </a>
</p>

**A fast HTTP logging server that catches every request! 🕵️‍♂️**

✨ **Perfect for webhook debugging and request monitoring** ✨

---

## 🚀 **Why Request Raccoon?**

- 🐳 **Zero Setup** - Works out of the box with Docker
- 📊 **Rich Logging** - JSON & text formats with full request details
- ⚡ **No Dependencies** - Pure Go, minimal footprint

## 🤔 What is it?

A Go-based HTTP server that logs all incoming requests 📝 Useful for debugging webhooks, monitoring API calls, and understanding request patterns 🔍

## ✨ Features

- 📊 Logs all HTTP requests (method, path, headers, query params, body)
- 🔒 Automatically redacts sensitive headers (Authorization, API keys, etc.)
- ⚙️ Configurable via environment variables
- 🐳 Docker support
- 📄 JSON or text log output
- 🚀 Zero external dependencies

## 🚀 Quick Start

### 🐳 Docker

```bash
# Basic usage
docker run -p 8080:8080 ghcr.io/czechbol/request-raccoon

# With custom configuration
docker run -p 3000:3000 -e PORT=3000 -e LOG_FORMAT=json ghcr.io/czechbol/request-raccoon
```

### 🔧 Go

```bash
git clone https://github.com/czechbol/request-raccoon.git
cd request-raccoon
go build -o request-raccoon cmd/http-logger/main.go
./request-raccoon
```

## ⚙️ Configuration

| Variable              | Default   | Description                          |
| --------------------- | --------- | ------------------------------------ |
| `PORT`                | `8080`    | Server port                          |
| `HOST`                | `0.0.0.0` | Server host                          |
| `LOG_LEVEL`           | `info`    | Log level (debug, info, warn, error) |
| `LOG_FORMAT`          | `text`    | Log format (text or json)            |
| `ENABLE_REQUEST_BODY` | `true`    | Log request bodies                   |

## 💡 Usage

Send requests to any path (except `/health`) 🎯

```bash
# Test webhook
curl http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{"event": "test"}'

# Test with query params
curl "http://localhost:8080/api/users?page=1&limit=10"
```

## 🛣️ Endpoints

- `GET /health` - 💚 Health check (not logged)
- `ANY /*` - 🎯 Universal handler (logs all requests)

## 📋 Log Output

### 📝 Text format

```
time=2024-01-15T10:30:00Z level=INFO msg="HTTP request received" method=POST path=/webhook
```

### 🔗 JSON format

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "HTTP request received",
  "method": "POST",
  "path": "/webhook",
  "headers": { "Authorization": "[REDACTED]" }
}
```

## 🔐 Security

Sensitive headers are automatically redacted 🙈

- Authorization
- Cookie/Set-Cookie
- X-API-Key
- X-Auth-Token
- Proxy-Authorization

## 👨‍💻 Development

### 📋 Prerequisites

- Go 1.24+ 🐹
- Docker (optional) 🐳

### 🛠️ Setup

```bash
git clone https://github.com/czechbol/request-raccoon.git
cd request-raccoon
go mod tidy
go test ./...
go build -o request-raccoon cmd/http-logger/main.go
./request-raccoon
```

### 📂 Project Structure

```
request-raccoon/
├── cmd/http-logger/     # Main application
├── internal/
│   ├── config/         # Configuration
│   ├── handler/        # Request handlers
│   ├── middleware/     # Logging middleware
│   └── server/         # HTTP server
└── Dockerfile          # Container config
```

## 🐳 Docker

Build your own image 🔨

```bash
docker build -t request-raccoon .
docker run -p 8080:8080 request-raccoon
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes and add tests
4. Run `go test ./...` and `golangci-lint run`
5. Submit a pull request

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details 📋
