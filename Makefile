.PHONY: dev run test test-front clean fix lint migrate-up migrate-down migrate-create api

# Disable Go workspace mode so -modfile (used by tools/go.mod) works correctly
# when this module is used as a submodule inside a go.work workspace.
export GOWORK=off

MIGRATE := go run -tags sqlite github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1
MIGRATIONS_PATH := internal/database/migrations
DB_PATH ?= ./data/wallydic.db
DATABASE_URL := sqlite://$(DB_PATH)

dev-back:
	air

dev-front:
	cd frontend && npm install &&  npm run dev

# Full production build
build: clean build-frontend build-backend
	@echo "✅ Build complete! Run ./bin/wallydic"

run: build
	./bin/wallydic

clean:
	rm -rf bin/
	rm -rf internal/ui/dist
	rm -rf frontend/dist

test: api-back
	go test -tags=test ./...

# Run frontend tests using the Node version from frontend/.nvmrc
test-front:
	cd frontend && node --version | grep -qE "^v$$(cat .nvmrc)" || \
		(echo "❌ Wrong Node version. Run: nvm use (expected v$$(cat .nvmrc))" && exit 1)
	cd frontend && npm test

# Run the linter (prints issues). Uses version from tools/go.mod.
lint: api-back
	go tool -modfile=tools/go.mod golangci-lint run ./...

# Run the linter and automatically fix what it can
fix: api-back
	go tool -modfile=tools/go.mod golangci-lint run --fix ./...

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

migrate-reapply-latest: migrate-down migrate-up

migrate-create:
	@read -p "Migration name: " name; \
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_PATH) -seq $$name

build-frontend: api-front
	@echo "📦 Building frontend..."
	cd frontend && npm install && npm run build
	@echo "📋 Copying dist to internal/ui..."
	rm -rf internal/ui/dist
	cp -r frontend/dist internal/ui/dist

build-backend: api-back
	@echo "🔨 Building Go binary..."
	go build -tags=prod -o bin/wallydic ./cmd/api

api-back:
	go generate ./...

api-front:
	cd frontend && npm run generate:api

api: api-back api-front
