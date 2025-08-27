# Slim
[![Slim CI](https://github.com/go-slim/slim/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/go-slim/slim/actions/workflows/ci.yml)

[中文](README.zh-CN.md)

A fast, composable HTTP framework for Go. Clean APIs, powerful routing, and pragmatic defaults.

Website: https://go-slim.dev

## Features

- Minimal core with pluggable middleware
- Fast radix-tree router with params and wildcards
- Centralized error handling with per-collector and router overrides
- Content negotiation (types, encodings, charsets, languages)
- JSON/XML/JSONP rendering and custom serializers
- Static files and directories via fs.FS
- Built-in middleware: logger, recovery, CORS, rate limiter
- Virtual hosting (vhost) support

## Installation

```bash
go get go-slim.dev/slim
```

## Quick Start

```go
package main

import (
    "net/http"
    "go-slim.dev/slim"
)

func main() {
    s := slim.New()

    s.GET("/hello", func(c slim.Context) error {
        return c.String(http.StatusOK, "Hello, Slim!")
    })

    s.Start(":8080")
}
```

## Examples

See runnable examples in `examples/`:
- `examples/static` — serve static files
- `examples/cors` — CORS middleware
- `examples/logger-recovery` — logging and panic recovery
- `examples/nego` — content negotiation
- `examples/rate-limiter` — rate limiting

Run any example:
```bash
cd examples/static
go run .
```

## Documentation

Full docs, guides, and API reference: https://go-slim.dev

## Testing & CI

Run tests locally:

```bash
go test ./...
```

Run benchmarks locally (examples):

```bash
# run all benchmarks with allocations
go test -run=^$ -bench . -benchmem

# run a specific benchmark
go test -run=^$ -bench BenchmarkResponse_StaticDir_LargeNested -benchmem
```

CI:

- GitHub Actions runs tests on every push and PR.
- Short benchmarks run in CI for `slim` with `-benchtime=100x`.

## License

Apache-2.0. See `LICENSE`.
