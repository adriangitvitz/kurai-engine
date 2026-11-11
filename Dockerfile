FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo dev) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /kurai ./cmd/kurai

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata poppler-utils

WORKDIR /app

COPY --from=builder /kurai .

COPY config.yaml .

RUN mkdir -p blocklists cache/blocklists data

RUN adduser -D -u 1000 kurai
RUN chown -R kurai:kurai /app/cache /app/data
USER kurai

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./kurai"]
CMD ["--config", "config.yaml"]
