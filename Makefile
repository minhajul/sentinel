build:
	docker-compose up -d --build

run-api:
	go run cmd/api/main.go

run-consumer:
	go run cmd/consumer/main.go

PWD=$(shell pwd)

migrate:
	docker run --rm -v $(PWD)/migrations:/migrations --network host migrate/migrate \
    		-path=/migrations/ \
    		-database "postgresql://user:password@localhost:5432/sentinel_db?sslmode=disable" up

# Standard stop (keeps data)
down:
	docker-compose down

# Full cleanup (deletes data & images)
clean:
	docker-compose down -v --rmi local --remove-orphans