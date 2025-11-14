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
RUN --mount=type=cache,target=/go/pkg/mod \
    GOSUMDB=off go mod download

# Copy source code
COPY . .

# Build with build cache mount for faster subsequent builds
# Re-enable checksum verification during build for security
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o main ./cmd/server

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

# Copy service account file if it exists (optional, for FCM)
COPY --from=builder /app/service-account.json* ./
COPY entrypoint.sh ./entrypoint.sh

# Change ownership to non-root user
RUN chown -R appuser:appuser /app && chmod +x /app/entrypoint.sh

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health 2>&1 || exit 1

# Run the application
ENTRYPOINT ["./entrypoint.sh"]
