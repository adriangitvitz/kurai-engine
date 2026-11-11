.PHONY: build run test test-cover cover-html bench lint clean docker-build setup up redeploy up-doh down help

BINARY := kurai
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/kurai

run: build
	./bin/$(BINARY) --config config.yaml

test:
	go test -v -race ./...

test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//' | \
	awk '{if ($$1 < 80) {print "Coverage " $$1 "% is below 80% threshold"; exit 1} else {print "Coverage: " $$1 "%"}}'

cover-html: test-cover
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

bench:
	go test -bench=. -benchmem ./internal/...

lint:
	golangci-lint run

clean:
	rm -rf bin/ coverage.out coverage.html

docker-build:
	docker build -t kurai:$(VERSION) .

setup:
	./scripts/setup.sh

up: setup
	docker compose up -d

redeploy: setup
	docker compose up -d --build

up-doh: setup
	docker compose --profile doh -f docker-compose.yml -f docker-compose.doh.yml up -d

down:
	docker compose down

help:
	@echo "Available targets:"
	@echo "  build           - Build the binary"
	@echo "  run             - Build and run locally"
	@echo "  test            - Run tests with race detector"
	@echo "  test-cover      - Run tests with coverage (fails if <80%)"
	@echo "  cover-html      - Generate HTML coverage report"
	@echo "  bench           - Run benchmarks"
	@echo "  lint            - Run linter"
	@echo "  clean           - Clean build artifacts"
	@echo "  docker-build    - Build Docker image"
	@echo "  setup           - Create .env and users.yaml, generate local secrets"
	@echo "  up              - Start with docker-compose"
	@echo "  redeploy        - Rebuild images + recreate (after code/.env changes)"
	@echo "  up-doh          - Start with DNS-over-HTTPS enabled"
	@echo "  down            - Stop docker-compose"
