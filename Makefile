
include .env
export

DB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)


.PHONY: run build test test-one lint docker-up docker-down migrate-up migrate-down migrate-create clean install-hooks fmt

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

test-one:
	go test ./... -run $(TEST)

lint:
	golangci-lint run ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate-up:
	migrate -path db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DB_URL)" down

migrate-create:
	migrate create -ext sql -dir db/migrations -seq $(NAME)

fmt:
	goimports -w .

clean:
	rm -rf bin/


install-hooks:
	git config core.hooksPath scripts/hooks
	chmod +x scripts/hooks/pre-commit
	@echo "Git hooks installed. Pre-commit checks are now active."