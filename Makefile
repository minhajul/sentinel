run-infra:
	docker-compose up -d

stop-infra:
	docker-compose down

run-api:
	go run cmd/api/main.go

run-consumer:
	go run cmd/consumer/main.go

migrate-up:
	migrate -path migrations -database "postgresql://user:password@localhost:5432/audit_db?sslmode=disable" up