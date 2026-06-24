.DEFAULT_GOAL := help
.PHONY: help \
        back-dev front-dev \
        back-test front-test \
        back-lint front-lint back-fix check \
        back-bench \
        back-seed back-seed-db back-seed-db-sample \
        api build build-debuggable clean \
        install-hooks release-patch release-minor release-major \
        _api-bundle _api-back _api-front _build-backend _build-frontend _check-migrations _release

# Disable Go workspace mode so -modfile (used by tools/go.mod) works correctly
# when this module is used as a submodule inside a go.work workspace.
export GOWORK=off

VERSION ?= $(shell (git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0) | sed 's/^v//')
NEXT_PATCH = $(shell echo $(VERSION) | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}')
NEXT_MINOR = $(shell echo $(VERSION) | awk -F. '{printf "%d.%d.0", $$1, $$2+1}')
NEXT_MAJOR = $(shell echo $(VERSION) | awk -F. '{printf "%d.0.0", $$1+1}')
SKIP_RELEASE_CHECK ?= 0

DB_PATH ?= ./data/data.db

help: ## Show this help
	@grep -E '^[a-zA-Z][a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Run locally
# ---------------------------------------------------------------------------

back-dev: ## Run the backend locally with hot reload (Air)
	air

front-dev: ## Run the frontend locally (Vite dev server)
	cd frontend && npm install && npm run dev

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

back-test: _api-back ## Run all Go tests (uses -tags=test)
	go test -tags=test ./cmd/... ./internal/...

# Run frontend tests using the Node version from frontend/.nvmrc
front-test: ## Run frontend tests (vitest)
	cd frontend && node --version | grep -qE "^v$$(cat .nvmrc)" || \
		(echo "❌ Wrong Node version. Run: nvm use (expected v$$(cat .nvmrc))" && exit 1)
	cd frontend && npm test

# ---------------------------------------------------------------------------
# Lint / checks
# ---------------------------------------------------------------------------

back-lint: _api-back _check-migrations ## Format + golangci-lint + migration-syntax check
	go tool -modfile=tools/go.mod golangci-lint run ./cmd/... ./internal/...

front-lint: ## ESLint + TypeScript type-check (tsconfig.app.json)
	cd frontend && npx eslint .
	cd frontend && npx tsc --noEmit -p tsconfig.app.json

# Run golangci-lint with --fix to auto-apply what it can
back-fix: _api-back ## golangci-lint with --fix (auto-fix what it can)
	go tool -modfile=tools/go.mod golangci-lint run --fix ./cmd/... ./internal/...

check: back-lint front-lint back-test front-test ## Full validation: lint + type-check + tests (back + front)

# ---------------------------------------------------------------------------
# Benchmarks
# ---------------------------------------------------------------------------

back-bench: _api-back ## Run all Go benchmarks (no tests)
	go test -tags=test -run='^$$' -bench=. -benchmem ./internal/...

# ---------------------------------------------------------------------------
# Database seeds
# ---------------------------------------------------------------------------

# Generate a clean, fully-seeded latest-schema SQLite DB → db-test-seeds/seed-<ts>.db.
# Reusable fixture for security audits, UX/manual testing and demos: copy the file
# to a stack's data/data.db (no migrations needed). Append-only; never deletes.
# Override row volume with SEED_ACCESS_LOG_VOLUME (default 250).
back-seed-db: _api-back ## Build a seed DB artifact (test/security world) → db-test-seeds/
	SEED_OUT_DIR=$(CURDIR)/db-test-seeds \
		go test -tags='test dbseed' -run TestGenerateSeedDB -count=1 ./internal/database/

# Like back-seed-db, but materialises the presentable sample world (SeedSampleWorld):
# recognizable services, named people, and 24h of diurnally-spread traffic that
# lights up the dashboard. Use for local dev, screenshots, walkthroughs and demos.
back-seed-db-sample: _api-back ## Build a seed DB artifact (presentable sample world) → db-test-seeds/
	SEED_OUT_DIR=$(CURDIR)/db-test-seeds SEED_WORLD=sample \
		go test -tags='test dbseed' -run TestGenerateSeedDB -count=1 ./internal/database/

# Seed the local dev DB ($(DB_PATH)) with the sample world, ready for `make back-dev`.
# Builds a fresh sample seed and plants the newest one as data.db (clearing any
# WAL/SHM sidecars). Stop back-dev first so the swap is safe.
back-seed: back-seed-db-sample ## Seed the local dev DB with the sample world
	@SEED=$$(ls -t $(CURDIR)/db-test-seeds/seed-*.db 2>/dev/null | head -1); \
	if [ -z "$$SEED" ]; then echo "❌ No seed produced in db-test-seeds/"; exit 1; fi; \
	echo "📦 Planting $$SEED → $(DB_PATH)..."; \
	mkdir -p $$(dirname $(DB_PATH)); \
	rm -f "$(DB_PATH)" "$(DB_PATH)-wal" "$(DB_PATH)-shm"; \
	cp "$$SEED" "$(DB_PATH)"; \
	echo "✅ Dev DB seeded. Login: admin / AdminPass123! — run 'make back-dev'."

# ---------------------------------------------------------------------------
# API codegen + build
# ---------------------------------------------------------------------------

api: _api-bundle _api-back _api-front ## Regenerate backend + frontend types from api/openapi.yaml

# Full production build → bin/pulseweaver
build: clean _build-frontend _build-backend ## Production build → bin/pulseweaver
	@echo "✅ Build complete! Run ./bin/pulseweaver"

# Production build with the loopback pprof debug server compiled in (127.0.0.1:6060).
# For local profiling against the prod-like stack ONLY — never release this binary.
build-debuggable: clean _build-frontend ## Production build with pprof debug server (local profiling only)
	@echo "🔨 Building Go binary with pprof debug server enabled..."
	go build -tags='prod pprof' -o bin/pulseweaver ./cmd/api
	@echo "✅ Debuggable build complete (pprof on 127.0.0.1:6060)! Run ./bin/pulseweaver"

clean: ## Remove build artifacts (bin/, dist/)
	rm -rf bin/
	rm -rf internal/ui/dist
	rm -rf frontend/dist

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

release-patch: ## Changelog → commit → tag → push (patch: x.y.Z+1)
	@$(MAKE) _release V=$(NEXT_PATCH)

release-minor: ## Changelog → commit → tag → push (minor: x.Y+1.0)
	@$(MAKE) _release V=$(NEXT_MINOR)

release-major: ## Changelog → commit → tag → push (major: X+1.0.0)
	@$(MAKE) _release V=$(NEXT_MAJOR)

# ---------------------------------------------------------------------------
# Internal helpers — prefixed with _ ; not meant to be run directly
# ---------------------------------------------------------------------------

_api-bundle:
	npm run bundle:api

_api-back: _api-bundle
	go generate ./cmd/... ./internal/...

_api-front: _api-bundle
	cd frontend && npm run generate:api

_build-frontend: _api-front
	@echo "📦 Building frontend..."
	cd frontend && npm install && npm run build
	@echo "📋 Copying dist to internal/ui..."
	rm -rf internal/ui/dist
	cp -r frontend/dist internal/ui/dist

_build-backend: _api-back
	@echo "🔨 Building Go binary..."
	go build -tags=prod -o bin/pulseweaver ./cmd/api

_check-migrations:
	@bash scripts/check-migrations.sh

# Run as $(MAKE) _release V=x.y.z — never call directly
_release:
	@git diff --quiet && git diff --staged --quiet || (echo "❌ Dirty working tree — commit or stash changes first" && exit 1)
	@echo "Current: v$(VERSION) → Next: v$(V)"
	@read -p "Confirm? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	@if [ "$(SKIP_RELEASE_CHECK)" != "1" ]; then $(MAKE) check; fi
	git-cliff --unreleased --tag "v$(V)" --prepend CHANGELOG.md
	@echo "Review and edit CHANGELOG.md before continuing"
	@read -p "Continue? [y/N] " confirm && [ "$$confirm" = "y" ] || exit 1
	git add CHANGELOG.md
	git diff --staged --quiet || git commit -m "chore: release v$(V)"
	git tag -a "v$(V)" -m "Release v$(V)"
	git push origin main --tags
