.PHONY: build up down clean run-api run-consumer test lint migrate

# Load .env into the recipe environment so values like POSTGRES_PASSWORD are
# available without the user having to `export` them manually.
ifneq (,$(wildcard .env))
include .env
export
endif

build:
	docker compose up -d --build

up:
	docker compose up -d

down:
	docker compose down

clean:
	docker compose down -v --rmi local --remove-orphans

run-api:
	go run cmd/api/main.go

run-consumer:
	go run cmd/consumer/main.go

test:
	go test ./... -count=1

lint:
	@command -v golangci-lint >/dev/null || { echo "golangci-lint not installed; see https://golangci-lint.run/welcome/install"; exit 1; }
	golangci-lint run ./...

PWD=$(shell pwd)

migrate:
	docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate \
    		-path=/migrations/ \
    		-database "postgresql://user:$(POSTGRES_PASSWORD)@localhost:5432/sentinel_db?sslmode=disable" up
