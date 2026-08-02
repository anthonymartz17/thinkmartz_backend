include .env
export

DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

.PHONY: run build test test-one lint mocks docker-up docker-down migrate-up migrate-down migrate-create clean install-hooks fmt

## --- Build & Run ---

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

## --- Test & Lint ---

test:
	go test ./...

test-one:
	go test ./... -run $(TEST)

lint:
	golangci-lint run ./...

mocks:
	mockgen -source=internal/auth/repository_interface.go \
		-destination=internal/auth/mocks/mock_repository.go \
		-package=mocks
	mockgen -source=internal/auth/tokenservice_interface.go \
		-destination=internal/auth/mocks/mock_tokenservice.go \
		-package=mocks

## --- Docker ---

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

## --- Migrations ---

migrate-up:
	migrate -path db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DB_URL)" down

migrate-create:
	migrate create -ext sql -dir db/migrations -seq $(NAME)

## --- Dev Tooling ---

fmt:
	goimports -w .

clean:
	rm -rf bin/

install-hooks:
	git config core.hooksPath scripts/hooks
	chmod +x scripts/hooks/pre-commit
	@echo "Git hooks installed. Pre-commit checks are now active."