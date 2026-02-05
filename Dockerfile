FROM golang:1.21-alpine AS go-builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev sqlite-dev

COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN go mod tidy && go mod download

ENV CGO_ENABLED=1
RUN go build -ldflags="-s -w" -o fleet-monitor ./cmd/

FROM alpine:3.19 AS cpp-builder

WORKDIR /build

RUN apk add --no-cache g++ make

COPY cpp-parser/ ./cpp-parser/

RUN cd cpp-parser && make

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache sqlite-libs libstdc++ curl python3

COPY --from=go-builder /build/fleet-monitor /app/
COPY --from=cpp-builder /build/cpp-parser/fleet_parser /app/
COPY data/ /app/data/
COPY web/ /app/web/
COPY scripts/ /app/scripts/

RUN mkdir -p /app/data && chmod +x /app/scripts/*.py

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost:8080/health || exit 1

CMD ["./fleet-monitor", "server", "--port", "8080"]
