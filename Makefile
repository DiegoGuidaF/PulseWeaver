.PHONY: dev run test clean fix migrate-up migrate-down migrate-create

dev-back:
	air

dev-front:
	cd frontend && npm run dev

run:
	go run cmd/api/main.go

test:
	go test -v ./...

fix:
	go fmt ./...
	golangci-lint run ./...

migrate-up:
	migrate -path internal/database/migrations -database "sqlite3://./data.db" up

migrate-down:
	migrate -path internal/database/migrations -database "sqlite3://./data.db" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/database/migrations -seq $$name