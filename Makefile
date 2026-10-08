.PHONY: build run test test-race lint fmt migrate-up migrate-down seed docker-up docker-down clean help

BINARY_API=forgestock-api
BINARY_WORKER=forgestock-worker
DOCKER_IMAGE=forgestock

help:
	@echo "Available commands:"
	@echo "  build        - Build the API and worker binaries"
	@echo "  run          - Run the API server locally"
	@echo "  run-worker   - Run the worker locally"
	@echo "  test         - Run unit tests"
	@echo "  test-race    - Run tests with race detector"
	@echo "  test-integration - Run integration tests"
	@echo "  lint         - Run golangci-lint"
	@echo "  fmt          - Format code with gofmt"
	@echo "  migrate-up   - Run database migrations up"
	@echo "  migrate-down - Run database migrations down"
	@echo "  seed         - Seed development data"
	@echo "  docker-up    - Start Docker Compose services"
	@echo "  docker-down  - Stop Docker Compose services"
	@echo "  docker-build - Build Docker images"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  generate     - Generate code (mocks, etc.)"

build:
	CGO_ENABLED=0 go build -o bin/$(BINARY_API) ./cmd/api
	CGO_ENABLED=0 go build -o bin/$(BINARY_WORKER) ./cmd/worker

run:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./... -count=1

test-race:
	go test -race ./... -count=1

test-integration:
	go test -tags=integration ./... -count=1

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

seed:
	go run ./scripts/seed

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f

clean:
	rm -rf bin/
	docker-compose down -v
	docker system prune -f

deps:
	go mod download
	go mod tidy

generate:
	go generate ./...

# Development helpers
dev-setup: deps docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@make migrate-up
	@make seed
	@echo "Development environment ready!"

dev-teardown: docker-down

# CI commands
ci-test: test-race
ci-lint: lint
ci-build: build