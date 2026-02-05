# =============================================================================
# Fleet Telemetry Monitor - Multi-stage Docker Build
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Build Go application
# -----------------------------------------------------------------------------
FROM golang:1.21-alpine AS go-builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files first for better caching
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build with CGO for SQLite support
ENV CGO_ENABLED=1
RUN go build -ldflags="-s -w" -o fleet-monitor ./cmd/

# -----------------------------------------------------------------------------
# Stage 2: Build C++ parser
# -----------------------------------------------------------------------------
FROM alpine:3.19 AS cpp-builder

WORKDIR /build

# Install build tools
RUN apk add --no-cache g++ make

# Copy C++ source
COPY cpp-parser/ ./cpp-parser/

# Build optimized parser
RUN cd cpp-parser && make

# -----------------------------------------------------------------------------
# Stage 3: Final runtime image
# -----------------------------------------------------------------------------
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache \
    sqlite-libs \
    libstdc++ \
    ca-certificates \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 fleet && \
    adduser -u 1000 -G fleet -s /bin/sh -D fleet

# Copy binaries from builders
COPY --from=go-builder /build/fleet-monitor /app/
COPY --from=cpp-builder /build/cpp-parser/fleet_parser /app/

# Copy static files and sample data
COPY data/ /app/data/
COPY scripts/ /app/scripts/
COPY web/ /app/web/

# Create data directory with proper permissions
RUN mkdir -p /app/data && \
    chown -R fleet:fleet /app

# Switch to non-root user
USER fleet

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Environment variables
ENV PORT=8080
ENV DB_PATH=/app/data/fleet_telemetry.db
ENV GIN_MODE=release

# Default command - start API server
CMD ["./fleet-monitor", "server", "--port", "8080", "--db", "/app/data/fleet_telemetry.db"]
