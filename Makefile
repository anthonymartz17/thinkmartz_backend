include .env
export

DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

.PHONY: run dev build test test-one lint mocks docker-up docker-down db-shell redis-shell migrate-up migrate-down migrate-create seed seed-down clean install-hooks fmt

## --- Build & Run ---

run:
	go run ./cmd/api

dev:
	air

build:
	go build -o bin/api ./cmd/api

## --- Test & Lint ---

test:
	go test ./...

test-one:
	go test ./... -run $(TEST) -v

lint:
	golangci-lint run ./...

mocks:
	mockgen -source=internal/auth/repository.go \
		-destination=internal/auth/mocks/mock_repository.go \
		-package=mocks
	mockgen -source=internal/auth/token_issuer.go \
		-destination=internal/auth/mocks/mock_tokenservice.go \
		-package=mocks
	mockgen -source=internal/auth/authenticator.go \
		-destination=internal/auth/mocks/mock_service.go \
		-package=mocks
	mockgen -source=internal/post/repository.go \
		-destination=internal/post/mocks/mock_repository.go \
		-package=mocks
	mockgen -source=internal/post/feed_writer.go \
		-destination=internal/post/mocks/mock_feed_writer.go \
		-package=mocks
	mockgen -source=internal/post/service.go \
		-destination=internal/post/mocks/mock_service.go \
		-package=mocks
	mockgen -source=internal/feed/repository.go \
		-destination=internal/feed/mocks/mock_repository.go \
		-package=mocks
	mockgen -source=internal/feed/cache.go \
		-destination=internal/feed/mocks/mock_cache.go \
		-package=mocks
	mockgen -source=internal/feed/reader.go \
		-destination=internal/feed/mocks/mock_reader.go \
		-package=mocks

## --- Docker ---

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

db-shell:
	docker exec -it thinkmartz_backend-postgres-1 psql -U postgres -d thinkmartz

redis-shell:
	docker exec -it thinkmartz_backend-redis-1 redis-cli

## --- Migrations ---

migrate-up:
	migrate -path db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DB_URL)" down

migrate-create:
	migrate create -ext sql -dir db/migrations -seq $(NAME)

## --- Seed Data ---

seed:
	psql "$(DB_URL)" -f db/seed.sql

seed-down:
	psql "$(DB_URL)" -c "DELETE FROM users WHERE email LIKE '%@seed.example';"

## --- Dev Tooling ---

fmt:
	goimports -w .

clean:
	rm -rf bin/ tmp/

install-hooks:
	git config core.hooksPath scripts/hooks
	chmod +x scripts/hooks/pre-commit
	@echo "Git hooks installed. Pre-commit checks are now active."