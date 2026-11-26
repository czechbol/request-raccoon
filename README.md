# 🦝 Request Raccoon

<p align="center">
  <a href="https://golang.org">
    <img src="https://img.shields.io/badge/Go-1.27+-blue?style=for-the-badge" alt="Go Version">
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

| Variable              | Default    | Description                                                   |
| --------------------- | ---------- | ------------------------------------------------------------- |
| `PORT`                | `8080`     | Server port                                                   |
| `HOST`                | `0.0.0.0`  | Server host                                                   |
| `LOG_LEVEL`           | `info`     | Log level (debug, info, warn, error)                          |
| `LOG_FORMAT`          | `text`     | Log format (text or json)                                     |
| `ENABLE_REQUEST_BODY` | `true`     | Log request bodies                                            |
| `MAX_LOGGED_BODY`     | `65536`    | Maximum request body bytes to log (capped at `10485760`)      |
| `MAX_REQUEST_BODY`    | `33554432` | Maximum accepted request body size in bytes (`0` = no limit)  |
| `READ_TIMEOUT`        | `60s`      | Time allowed to read a request, as a Go duration (`0` = none) |
| `ENDPOINTS`           | none       | [Custom endpoints](#custom-endpoints), as a JSON array        |

Request bodies larger than `MAX_LOGGED_BODY` are logged truncated (without splitting a UTF-8 character), with
`request_body_truncated=true` and, when the `Content-Length` is known, `request_body_size`. With `0`, every non-empty
body is logged as empty and marked as truncated. Negative or non-numeric values use the default, and values above
10 MiB are lowered to 10 MiB.

### Request limits

Both limits are enabled by default and apply to every path, including `/health`.

- A body larger than `MAX_REQUEST_BODY` (32 MiB by default) is rejected with `413 Request Entity Too Large`, with or
  without a `Content-Length`. The rejection happens once more bytes than the limit have been received.
- A body that is still being read when `READ_TIMEOUT` expires is rejected with `408 Request Timeout`. The time is
  counted for the whole request, headers included (headers must also arrive within 10 seconds). It is also how long an
  idle keep-alive connection stays open.

For both variables, negative or unparsable values use the default. `READ_TIMEOUT` needs a unit, such as `90s` or `5m`.

The router redirects unclean paths, such as `//webhook`, without reading the rest of the body, so such a request can
receive the redirect instead of `413` or `408`.

### Unreadable request bodies

Rejected requests are logged like any other. How a read failure is reported and answered depends on when it happens:

| Failure happens                           | Log output                                     | Too large | Too slow | Any other failure |
| ----------------------------------------- | ---------------------------------------------- | --------- | -------- | ----------------- |
| While the logged part of the body is read | `request_body_error` field in the request line | `413`     | `408`    | `400`             |
| After the request line has been logged    | Separate `Failed to read request body` warning | `413`     | `408`    | `200`             |

The logged part is the first `MAX_LOGGED_BODY` bytes plus one. When reading it fails, `request_body` holds what was
received, if anything. With `ENABLE_REQUEST_BODY=false`, every failure happens after the request line.

### Custom endpoints

`ENDPOINTS` makes matching requests receive your own status, headers and body. All other requests still get the default
`200` response.

```bash
docker run -p 8080:8080 \
  -e ENDPOINTS='[
    {"pattern":"POST /webhook/github","status":202,"body":"accepted"},
    {"pattern":"/fail","status":503,"body":"try later"}
  ]' \
  ghcr.io/czechbol/request-raccoon
```

In a Compose file, use a block value. Write a literal `$` as `$$`.

```yaml
environment:
  ENDPOINTS: |
    [
      {
        "pattern": "GET /users/{id}",
        "status": 404,
        "headers": {"Content-Type": "application/json"},
        "body": "{\"error\": \"not found\"}"
      }
    ]
```

| Field     | Required | Default | Description                                                                 |
| --------- | -------- | ------- | --------------------------------------------------------------------------- |
| `pattern` | yes      |         | A Go [`ServeMux` pattern](https://pkg.go.dev/net/http#hdr-Patterns-ServeMux) |
| `status`  | no       | `200`   | Response status, from 200 to 599                                            |
| `headers` | no       | none    | Response headers, except `Content-Length` and `Transfer-Encoding`           |
| `body`    | no       | empty   | Response body; not allowed with status 204 or 304                           |

- A pattern without a method matches every method. `GET` patterns also match `HEAD`. A pattern ending in `/` matches
  everything below it.
- Without a `Content-Type` header, the type of a body is guessed from its content.
- Requests are logged, and the [request limits](#request-limits) apply, as for any other path.
- The server does not start if the value is invalid, if patterns conflict, or if an endpoint uses `/health` or `/`.
  Use `/{$}` to match only the root path.
- The startup log lists patterns and statuses, not headers or bodies.

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

- `GET /health` - 💚 Health check
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

- Go 1.27+ 🐹
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

### 🧬 Mutation testing

[Gremlins](https://gremlins.dev) changes the code and checks that the tests notice. CI fails when less than 90% of the
tested changes are caught. A change reported as `LIVED` was not caught.

```bash
go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
gremlins unleash --integration --coverpkg ./... --timeout-coefficient 5 \
  --invert-assignments --invert-logical --invert-loopctrl \
  --invert-bitwise --invert-bwassign --remove-self-assignments
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
