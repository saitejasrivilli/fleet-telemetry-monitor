# Build Go application
FROM golang:1.21-alpine AS go-builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy ALL source files first
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Download dependencies (this will read imports from source)
RUN go mod tidy
RUN go mod download

# Build with CGO for SQLite support
ENV CGO_ENABLED=1
RUN go build -ldflags="-s -w" -o fleet-monitor ./cmd/

# Build C++ parser
FROM alpine:3.19 AS cpp-builder

WORKDIR /build

RUN apk add --no-cache g++ make

COPY cpp-parser/ ./cpp-parser/

RUN cd cpp-parser && make

# Final runtime image
FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache sqlite-libs libstdc++ curl

COPY --from=go-builder /build/fleet-monitor /app/
COPY --from=cpp-builder /build/cpp-parser/fleet_parser /app/
COPY data/ /app/data/
COPY web/ /app/web/

RUN mkdir -p /app/data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost:8080/health || exit 1

CMD ["./fleet-monitor", "server", "--port", "8080"]
