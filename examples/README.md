# Slim Examples

This directory contains runnable example apps demonstrating common features:

- CORS: `examples/cors/`
- Rate Limiter: `examples/rate-limiter/`
- Logger + Recovery: `examples/logger-recovery/`
- Content Negotiation: `examples/nego/`
- Static files: `examples/static/`

Run one example at a time (each uses a unique port). In a separate terminal, use curl commands below to try them.

## Makefile 快速命令

在仓库根目录运行（或进入 `slim/examples/` 后去掉 `-C slim/examples` 部分）：

```bash
# 本地运行
make -C slim/examples run-cors
make -C slim/examples run-rate-limiter
make -C slim/examples run-logger-recovery
make -C slim/examples run-nego
make -C slim/examples run-static

# 本地构建二进制（输出到 slim/examples/bin/）
make -C slim/examples build-cors
make -C slim/examples build-rate-limiter
make -C slim/examples build-logger-recovery
make -C slim/examples build-nego
make -C slim/examples build-static

# Docker 运行（构建并运行，端口已映射）
make -C slim/examples docker-cors
make -C slim/examples docker-rate-limiter
make -C slim/examples docker-logger-recovery
make -C slim/examples docker-nego
make -C slim/examples docker-static   # 会挂载 ./public 到容器 /app/public
```

## CORS (port :1325)

Start:

```bash
go run ./slim/examples/cors
```

Test simple request:

```bash
curl -i -H 'Origin: http://localhost:3000' http://127.0.0.1:1325/hello
```

Preflight:

```bash
curl -i -X OPTIONS \
  -H 'Origin: http://localhost:3000' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: Content-Type, Authorization' \
  http://127.0.0.1:1325/hello
```

## Rate Limiter (port :1326)

Start:

```bash
go run ./slim/examples/rate-limiter
```

Make two quick requests (second should be 429 Too Many Requests):

```bash
curl -i http://127.0.0.1:1326/
sleep 0.1
curl -i http://127.0.0.1:1326/
```

## Logger + Recovery (port :1327)

Start:

```bash
go run ./slim/examples/logger-recovery
```

Requests:

```bash
curl -i http://127.0.0.1:1327/ok
curl -i http://127.0.0.1:1327/panic
```

Observe logs in the server terminal; recovery will return 500 for `/panic` and print stack traces.

## Content Negotiation (port :1328)

Start:

```bash
go run ./slim/examples/nego
```

JSON (default):

```bash
curl -i -H 'Accept: application/json' http://127.0.0.1:1328/data
```

XML:

```bash
curl -i -H 'Accept: application/xml' http://127.0.0.1:1328/data
```

Wildcard:

```bash
curl -i -H 'Accept: */*' http://127.0.0.1:1328/data
```

## Static files (port :1329)

Start:

```bash
go run ./slim/examples/static
```

Prepare a public directory with an index:

```bash
mkdir -p public
printf 'index-ok' > public/index.html
```

Test:

```bash
curl -i http://127.0.0.1:1329/
curl -i http://127.0.0.1:1329/ping
```

使用 Docker 运行该示例时，建议在仓库根目录准备 `public/`（Makefile 已将其挂载到容器）：

```bash
make -C slim/examples docker-static
```

# Notes

- Examples import packages from the local module path `go-slim.dev/slim`.
- If you change ports, update the curl commands accordingly.
- Run only one example per port at a time.
