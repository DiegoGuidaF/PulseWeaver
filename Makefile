# golangci-lint version (keep in sync with .github/workflows/ci.yml)
GOLANGCI_LINT_VERSION := v2.9.0
GOLANGCI_LINT_RUN := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: dev run test clean fix lint migrate-up migrate-down migrate-create

dev-back:
	air

dev-front:
	cd frontend && npm run dev


# Full production build
build: clean build-frontend build-backend
	@echo "✅ Build complete! Run ./bin/wallydic"

run: build
	./bin/wallydic

clean:
	rm -rf bin/
	rm -rf internal/ui/dist
	rm -rf frontend/dist


test:
	go test -tags=test -v ./...

lint:
	go fmt ./...
	$(GOLANGCI_LINT_RUN) run ./...

migrate-up:
	migrate -path internal/database/migrations -database "sqlite://./data.db" up

migrate-down:
	migrate -path internal/database/migrations -database "sqlite://./data.db" down 1

migrate-reapply-latest: migrate-down migrate-up

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/database/migrations -seq $$name

build-frontend:
	@echo "📦 Building frontend..."
	cd frontend && npm install && npm run build
	@echo "📋 Copying dist to internal/ui..."
	rm -rf internal/ui/dist
	cp -r frontend/dist internal/ui/dist

build-backend:
	@echo "🔨 Building Go binary..."
	go build -o bin/wallydic ./cmd/api
