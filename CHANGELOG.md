# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
