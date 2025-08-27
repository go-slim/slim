# syntax=docker/dockerfile:1
FROM golang:1.22-alpine as builder
WORKDIR /app

# Speeds up downloads; add tools as needed
RUN apk add --no-cache ca-certificates tzdata

# Copy go.mod/go.sum for all modules at repo root
COPY go.work go.work.sum ./
COPY slim/go.mod slim/go.sum ./slim/
COPY slim/middleware/go.mod slim/middleware/go.sum ./slim/middleware/ 2>/dev/null || true
COPY slim/nego/go.mod slim/nego/go.sum ./slim/nego/ 2>/dev/null || true
COPY slim/serde/go.mod slim/serde/go.sum ./slim/serde/ 2>/dev/null || true

# Copy source
COPY . .

# Select example to build via build-arg
ARG EXAMPLE=slim/examples/cors
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /app/app ./${EXAMPLE}

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/app /app/app
# Static example expects ./public; mount a volume in Makefile
EXPOSE 1325 1326 1327 1328 1329
CMD ["/app/app"]
