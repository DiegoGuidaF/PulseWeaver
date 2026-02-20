# Tools are isolated in tools/go.mod to avoid polluting main module dependencies.
# Tools are run with -modfile=tools/go.mod to use the separate module.

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

# Run the linter (prints issues). Uses version from tools/go.mod.
lint:
	go fmt ./...
	go tool -modfile=tools/go.mod golangci-lint run ./...

# Run the linter and automatically fix what it can (gofmt, goimports, etc.).
fix:
	go tool -modfile=tools/go.mod golangci-lint run --fix ./...

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
	go build -tags=prod -o bin/wallydic ./cmd/api
