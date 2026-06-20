.PHONY: dev run test test-front seed-db seed-db-showcase clean fix lint lint-front typecheck-front lint-all check migrate-up migrate-down migrate-create api \
        install-hooks version release-patch release-minor release-major _release check-migrations

# Disable Go workspace mode so -modfile (used by tools/go.mod) works correctly
# when this module is used as a submodule inside a go.work workspace.
export GOWORK=off

VERSION ?= $(shell (git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0) | sed 's/^v//')
NEXT_PATCH = $(shell echo $(VERSION) | awk -F. '{printf "%d.%d.%d", $$1, $$2, $$3+1}')
NEXT_MINOR = $(shell echo $(VERSION) | awk -F. '{printf "%d.%d.0", $$1, $$2+1}')
NEXT_MAJOR = $(shell echo $(VERSION) | awk -F. '{printf "%d.0.0", $$1+1}')
SKIP_RELEASE_CHECK ?= 0

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

# Generate a clean, fully-seeded latest-schema SQLite DB → db-test-seeds/seed-<ts>.db.
# Reusable fixture for security audits, UX/manual testing and demos: copy the file
# to a stack's data/data.db (no migrations needed). Append-only; never deletes.
# Override row volume with SEED_ACCESS_LOG_VOLUME (default 250).
seed-db: api-back
	SEED_OUT_DIR=$(CURDIR)/db-test-seeds \
		go test -tags='test dbseed' -run TestGenerateSeedDB -count=1 ./internal/database/

# Like seed-db, but materialises the presentable demo world (SeedShowcaseWorld):
# recognizable services, named people, and 24h of diurnally-spread traffic that
# lights up the dashboard. Use for screenshots, walkthroughs and demos.
seed-db-showcase: api-back
	SEED_OUT_DIR=$(CURDIR)/db-test-seeds SEED_WORLD=showcase \
		go test -tags='test dbseed' -run TestGenerateSeedDB -count=1 ./internal/database/

# Run frontend tests using the Node version from frontend/.nvmrc
test-front:
	cd frontend && node --version | grep -qE "^v$$(cat .nvmrc)" || \
		(echo "❌ Wrong Node version. Run: nvm use (expected v$$(cat .nvmrc))" && exit 1)
	cd frontend && npm test

# Run the linter (prints issues). Uses version from tools/go.mod.
lint-back: api-back check-migrations
	go tool -modfile=tools/go.mod golangci-lint run ./cmd/... ./internal/...

# Run frontend ESLint
lint-front:
	cd frontend && npx eslint .

# Run frontend TypeScript type-check (uses tsconfig.app.json, not the root solution config)
typecheck-front:
	cd frontend && npx tsc --noEmit -p tsconfig.app.json

# Run all linters and type-checks (backend + frontend)
lint-all: lint-back lint-front typecheck-front

# Full pre-push check: lint + typecheck + test everything
check: check-migrations lint-all test test-front

# Run the linter and automatically fix what it can
fix: api-back
	go tool -modfile=tools/go.mod golangci-lint run --fix ./cmd/... ./internal/...

migrate-up:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

migrate-reapply-latest: migrate-down migrate-up

migrate-create:
	@if [ -n "$(NAME)" ]; then \
		$(MIGRATE) create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME); \
	else \
		read -p "Migration name: " name; \
		$(MIGRATE) create -ext sql -dir $(MIGRATIONS_PATH) -seq $$name; \
	fi

check-migrations: ## Verify all migration files have explicit BEGIN TRANSACTION / COMMIT
	@bash scripts/check-migrations.sh

build-frontend: api-front
	@echo "📦 Building frontend..."
	cd frontend && npm install && npm run build
	@echo "📋 Copying dist to internal/ui..."
	rm -rf internal/ui/dist
	cp -r frontend/dist internal/ui/dist

build-backend: api-back
	@echo "🔨 Building Go binary..."
	go build -tags=prod -o bin/pulseweaver ./cmd/api

api-back: api-bundle
	go generate ./cmd/... ./internal/...

api-front: api-bundle
	cd frontend && npm run generate:api

api-bundle:
	npm run bundle:api

api: api-bundle api-back api-front

gh-auto-merge-dependabot:
	gh pr list --repo DiegoGuidaF/PulseWeaver --author "app/dependabot" --state open --json number,title,statusCheckRollup --jq '.[] | select([.statusCheckRollup[].conclusion] | all(. == "SUCCESS" or . == "SKIPPED")) | .number' | xargs -I{} gh pr merge {} --squash --delete-branch --repo DiegoGuidaF/PulseWeaver

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

release-patch: ## Changelog → commit → tag → push (patch: x.y.Z+1)
	@$(MAKE) _release V=$(NEXT_PATCH)

release-minor: ## Changelog → commit → tag → push (minor: x.Y+1.0)
	@$(MAKE) _release V=$(NEXT_MINOR)

release-major: ## Changelog → commit → tag → push (major: X+1.0.0)
	@$(MAKE) _release V=$(NEXT_MAJOR)

# Internal: run as $(MAKE) _release V=x.y.z — never call directly
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
