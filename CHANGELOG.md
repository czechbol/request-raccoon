# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `MAX_LOGGED_BODY` setting for the maximum number of request body bytes to log (default 65536, capped at 10 MiB)
- `request_body_truncated` and `request_body_size` log fields for request bodies over the limit
- Warning with the request path when reading the rest of a request body fails after the request has been logged
- `MAX_REQUEST_BODY` setting for the maximum accepted request body size in bytes (default 32 MiB, `0` for no limit)
- `READ_TIMEOUT` setting for the time allowed to read a request (default `60s`, `0` for none)

### Changed

- Go 1.27 is now required (built with 1.27.4)
- Updated the GitHub Actions and gosec used by CI and releases
- Request bodies over the limit are logged truncated instead of being omitted, and the limit is raised from 1024 bytes to 65536 bytes by default
- **Requests that were previously accepted can now be rejected:** a request body over 32 MiB is answered with 413 Request Entity Too Large, and one that takes longer than 60 seconds to read with 408 Request Timeout, including on `/health`. Raise the limits or set them to `0` to disable them
- Idle keep-alive connections are closed after `READ_TIMEOUT` (60 seconds by default)
- A request body that fails while its logged part is being read is answered with 413 when too large, 408 when too slow and 400 Bad Request otherwise, instead of 500 Internal Server Error
- A request body that fails after the request has been logged, which includes every failure with `ENABLE_REQUEST_BODY=false`, is answered with 413 when too large and 408 when too slow; other such failures are still answered with 200 OK

### Fixed

- Memory used for request body logging no longer grows with the size of the body
- Requests whose body fails while its logged part is being read are still logged, with the partial body and a `request_body_error` field

## [1.0.0] - 2025-06-05

### Added

- 🦝 Initial release of Request Raccoon HTTP logging server
- 📊 HTTP request logging with method, path, headers, query parameters, and body
- 🔒 Automatic redaction of sensitive headers (Authorization, Cookie, Set-Cookie, X-API-Key, X-Auth-Token, Proxy-Authorization)
- ⚙️ Environment-based configuration system
- 🌐 Configurable server port and host
- 📝 Configurable log level (debug, info, warn, error)
- 📄 Configurable log format (text or JSON)
- 🔄 Toggle for request body logging
- 🐳 Docker support with multi-stage build
- 💚 Health check endpoint at `/health`
- 🎯 Universal request handler for all other paths
- 🧪 Comprehensive test coverage for all components
- 🔍 Go linting configuration with golangci-lint
- 📖 Complete documentation with usage examples
- 📄 MIT License

[Unreleased]: https://github.com/czechbol/request-raccoon/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/czechbol/request-raccoon/releases/tag/v1.0.0
