.PHONY: help build run test clean fmt lint docker-build docker-run migrate-up migrate-down migrate-create dev

help:
	@echo "Available commands:"
	@echo "  make build        - Build the binary"
	@echo "  make run          - Run the server"
	@echo "  make test         - Run tests"
	@echo "  make test-cover   - Run tests with coverage report"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linter"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run Docker container"

build:
	go build -o bin/server cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test -v ./...

test-cover:
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path internal/database/migrations -database "sqlite3://./data.db" up

migrate-down:
	migrate -path internal/database/migrations -database "sqlite3://./data.db" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/database/migrations -seq $$name

dev:
	air