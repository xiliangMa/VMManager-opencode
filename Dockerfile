# ============================================
# VMManager - Multi-Architecture Dockerfile
# ============================================

# -------------------
# Build Stage
# -------------------
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install git for go modules
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && \
    go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w -X 'vmmanager/internal/version.Version=$(git describe --tags --always 2>/dev/null || echo 'dev')'" \
    -o server ./cmd/server/

# -------------------
# Runtime Stage
# -------------------
FROM alpine:latest AS runtime

ARG VERSION=dev

# Install runtime dependencies
RUN apk --no-cache add \
    libvirt-daemon \
    qemu-img \
    ca-certificates \
    gettext \
    tzdata

# Create non-root user
RUN addgroup -g 1000 app && \
    adduser -u 1000 -G app -s /bin/sh -D app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy configuration and data
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/uploads ./uploads
COPY --from=builder /app/translations ./translations
COPY --from=builder /app/docs ./docs

# Create directories with proper permissions
RUN mkdir -p /app/data /app/logs && \
    chown -R app:app /app && \
    chmod +x /app/server

# Switch to non-root user
USER app

# Expose ports
EXPOSE 8080 8081

# Set environment variables
ENV APP_VERSION=$VERSION
ENV GIN_MODE=release

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/health || exit 1

# Run the application
CMD ["./server"]

# ============================================
# Multi-platform build example:
# ============================================
# docker buildx build --platform linux/amd64,linux/arm64 -t vmmanager/server:latest --push .
# ============================================
