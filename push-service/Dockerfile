FROM golang:1.24-alpine AS builder

# Configure Go to use multiple proxies and optimize download settings
ENV GOPROXY=https://proxy.golang.org,https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.org
ENV GOTIMEOUT=300s
ENV GOPRIVATE=

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies with cache mount (BuildKit feature for faster builds)
# Disable checksum verification for faster downloads (re-enabled in build step)
RUN --mount=type=cache,id=go-mod-cache,target=/go/pkg/mod \
    GOSUMDB=off go mod download

# Copy source code
COPY . .

# Build with build cache mount for faster subsequent builds
# Re-enable checksum verification during build for security
RUN --mount=type=cache,id=go-mod-cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o main ./cmd/server

# Install migrate CLI for runtime database migrations (with Postgres driver)
RUN --mount=type=cache,id=go-mod-cache,target=/go/pkg/mod \
    --mount=type=cache,id=go-build-cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.0

# Final stage
FROM alpine:latest

# Install ca-certificates, wget (healthcheck) and netcat (dependency wait)
RUN apk --no-cache add ca-certificates tzdata wget netcat-openbsd

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

# Copy binary and config files
COPY --from=builder /app/main .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/migrations ./migrations

# Copy service account file if it exists (optional, for FCM)
COPY --from=builder /app/service-account.json* ./
COPY entrypoint.sh ./entrypoint.sh
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate

# Change ownership to non-root user
RUN chown -R appuser:appuser /app && chmod +x /app/entrypoint.sh && chmod +x /usr/local/bin/migrate

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health 2>&1 || exit 1

# Run the application
ENTRYPOINT ["./entrypoint.sh"]
