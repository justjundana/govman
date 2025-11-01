# Dockerfile for GOVMAN
# Multi-stage build for minimal image size

# Build stage
FROM golang:1.25-alpine AS builder

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build binary with optimizations and version info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w \
    -X 'github.com/justjundana/govman/internal/version.Version=${VERSION}' \
    -X 'github.com/justjundana/govman/internal/version.Commit=${COMMIT}' \
    -X 'github.com/justjundana/govman/internal/version.Date=${DATE}'" \
    -a -installsuffix cgo \
    -trimpath \
    -o govman \
    ./cmd/govman

# Verify binary was built
RUN test -f govman && chmod +x govman

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    git \
    curl \
    tar \
    gzip \
    bash \
    && adduser -D -s /bin/sh govman

# Copy timezone data from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from builder
COPY --from=builder /app/govman /usr/local/bin/govman

# Create govman directories with proper permissions
RUN mkdir -p /home/govman/.govman/{bin,cache,versions,downloads} && \
    chown -R govman:govman /home/govman/.govman && \
    chmod -R 755 /home/govman/.govman

# Switch to non-root user for security
USER govman
WORKDIR /home/govman

# Set environment variables
ENV HOME=/home/govman
ENV PATH="/home/govman/.govman/bin:${PATH}"
ENV GOVMAN_HOME="/home/govman/.govman"

# Health check - verify govman is working
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD govman version || exit 1

# Default entrypoint and command
ENTRYPOINT ["govman"]
CMD ["--help"]

# Metadata labels (OCI standard)
LABEL org.opencontainers.image.title="GOVMAN - Go Version Manager"
LABEL org.opencontainers.image.description="Cross-platform Go version manager for easy installation and switching between Go versions"
LABEL org.opencontainers.image.url="https://github.com/justjundana/govman"
LABEL org.opencontainers.image.source="https://github.com/justjundana/govman"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.vendor="justjundana"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${DATE}"
LABEL org.opencontainers.image.revision="${COMMIT}"