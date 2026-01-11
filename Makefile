build:
	docker-compose up -d --build

down:
	docker-compose down

run-api:
	go run cmd/api/main.go

run-consumer:
	go run cmd/consumer/main.go

PWD=$(shell pwd)

migrate-up:
	docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate \
    		-path=/migrations/ \
    		-database "postgresql://user:password@localhost:5432/sentinel_db?sslmode=disable" up