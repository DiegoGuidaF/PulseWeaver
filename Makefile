.PHONY: dev run test test-front clean fix lint lint-front typecheck-front lint-all check migrate-up migrate-down migrate-create api \
        install-hooks version release-patch release-minor release-major

# Disable Go workspace mode so -modfile (used by tools/go.mod) works correctly
# when this module is used as a submodule inside a go.work workspace.
export GOWORK=off

VERSION ?= $(shell (git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0) | sed 's/^v//')
NEXT_PATCH = $(shell echo $(VERSION) | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}')
NEXT_MINOR = $(shell echo $(VERSION) | awk -F. '{printf "%d.%d.0", $$1, $$2+1}')
NEXT_MAJOR = $(shell echo $(VERSION) | awk -F. '{printf "%d.0.0", $$1+1}')

MIGRATE := go run -tags sqlite github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1
MIGRATIONS_PATH := internal/database/migrations
DB_PATH ?= ./data/data.db
DATABASE_URL := sqlite://$(DB_PATH)

dev-back:
	air

dev-front:
	cd frontend && npm install &&  npm run dev

# Full production build
build: clean build-frontend build-backend
	@echo "✅ Build complete! Run ./bin/pulseweaver"

run: build
	./bin/pulseweaver

clean:
	rm -rf bin/
	rm -rf internal/ui/dist
	rm -rf frontend/dist

test: api-back
	go test -tags=test ./cmd/... ./internal/...

# Run frontend tests using the Node version from frontend/.nvmrc
test-front:
	cd frontend && node --version | grep -qE "^v$$(cat .nvmrc)" || \
		(echo "❌ Wrong Node version. Run: nvm use (expected v$$(cat .nvmrc))" && exit 1)
	cd frontend && npm test

# Run the linter (prints issues). Uses version from tools/go.mod.
lint: api-back
	go tool -modfile=tools/go.mod golangci-lint run ./cmd/... ./internal/...

# Run frontend ESLint
lint-front:
	cd frontend && npx eslint .

# Run frontend TypeScript type-check (uses tsconfig.app.json, not the root solution config)
typecheck-front:
	cd frontend && npx tsc --noEmit -p tsconfig.app.json

# Run all linters and type-checks (backend + frontend)
lint-all: lint lint-front typecheck-front

# Full pre-push check: lint + typecheck + test everything
check: lint-all test test-front

# Run the linter and automatically fix what it can
fix: api-back
	go tool -modfile=tools/go.mod golangci-lint run --fix ./cmd/... ./internal/...

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
	go build -tags=prod -o bin/pulseweaver ./cmd/api

api-back:
	go generate ./cmd/... ./internal/...

api-front:
	cd frontend && npm run generate:api

api: api-back api-front

# ---------------------------------------------------------------------------
# Hooks
# ---------------------------------------------------------------------------

install-hooks: ## Install git hooks (run once after cloning)
	@ln -sf "$(PWD)/scripts/commit-msg" "$(PWD)/.git/hooks/commit-msg"
	@chmod +x "$(PWD)/.git/hooks/commit-msg"
	@echo "✅ Git hooks installed."

# ---------------------------------------------------------------------------
# Release
# ---------------------------------------------------------------------------

version: ## Show current version
	@echo "v$(VERSION)"

release-patch: ## Tag and push a patch release (x.y.Z+1)
	@echo "Current: v$(VERSION) → Next: v$(NEXT_PATCH)"
	@read -p "Confirm? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	git tag -a "v$(NEXT_PATCH)" -m "Release v$(NEXT_PATCH)"
	git push origin main --tags

release-minor: ## Tag and push a minor release (x.Y+1.0)
	@echo "Current: v$(VERSION) → Next: v$(NEXT_MINOR)"
	@read -p "Confirm? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	git tag -a "v$(NEXT_MINOR)" -m "Release v$(NEXT_MINOR)"
	git push origin main --tags

release-major: ## Tag and push a major release (X+1.0.0)
	@echo "Current: v$(VERSION) → Next: v$(NEXT_MAJOR)"
	@read -p "Confirm? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	git tag -a "v$(NEXT_MAJOR)" -m "Release v$(NEXT_MAJOR)"
	git push origin main --tags
