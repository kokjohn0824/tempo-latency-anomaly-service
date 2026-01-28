## Multi-stage Dockerfile for Tempo Latency Anomaly Service
## Stage 1: Build
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum* ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Install swag for generating Swagger docs
RUN --mount=type=cache,target=/go/pkg/mod \
    go install github.com/swaggo/swag/cmd/swag@latest

# Copy the rest of the source
COPY . .

# Generate Swagger documentation
RUN /go/bin/swag init -g cmd/server/main.go -o ./docs

# Build static linux binary
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w" -o /out/tempo-anomaly ./cmd/server

## Stage 2: Runtime (small)
FROM alpine:3.19 AS runtime
RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S app && adduser -S -g app app

ENV TZ=Asia/Taipei \
    CONFIG_FILE=/config/config.yaml

WORKDIR /app

# Default config path; allow override via volume/env
COPY --from=builder /out/tempo-anomaly /app/tempo-anomaly
COPY --from=builder /src/configs/config.dev.yaml /config/config.yaml

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -fsS http://127.0.0.1:8080/healthz || exit 1

USER app

ENTRYPOINT ["/app/tempo-anomaly"]
