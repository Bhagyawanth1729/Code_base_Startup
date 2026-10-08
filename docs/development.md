# Development Guide

## Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Redis 7+
- golang-migrate
- Docker & Docker Compose (optional)
- Make

## Initial Setup

```bash
# Clone repository
git clone <repo-url>
cd forgestock

# Copy environment template
cp .env.example .env

# Edit .env with your settings
# Required: JWT_SECRET, DATABASE_URL, REDIS_URL

# Download dependencies
make deps

# Start dependencies (PostgreSQL, Redis)
make docker-up

# Run migrations
make migrate-up

# Seed development data
make seed

# Run API server
make run

# In another terminal, run worker
make run-worker
```

## Project Structure

```
forge-stock/
├── cmd/
│   ├── api/main.go       # HTTP server entry point
│   └── worker/main.go    # Background worker entry point
├── internal/
│   ├── auth/             # Authentication & authorization
│   ├── users/            # User management
│   ├── products/         # Product management
│   ├── categories/       # Category management
│   ├── inventory/        # Inventory management
│   ├── orders/           # Order management
│   ├── payments/         # Payment processing
│   ├── notifications/    # Notification system
│   ├── audit/            # Audit logging
│   ├── health/           # Health checks
│   └── platform/         # Shared infrastructure
│       ├── database/     # PostgreSQL
│       ├── redis/        # Redis
│       ├── http/         # HTTP utilities
│       ├── logger/       # Logging
│       ├── metrics/      # Prometheus metrics
│       └── config/       # Configuration
├── migrations/           # Database migrations
├── tests/                # Integration tests
├── scripts/              # Operational scripts
├── deployments/          # Docker, K8s configs
└── docs/                 # Documentation
```

## Adding a New Module

1. Create module directory:
   ```bash
   mkdir internal/newmodule
   ```

2. Create files:
   ```
   internal/newmodule/
   ├── handler.go
   ├── service.go
   ├── repository.go
   ├── models.go
   └── *_test.go
   ```

3. Follow existing patterns:
   - Repository: data access only
   - Service: business logic
   - Handler: HTTP concerns only
   - Use dependency injection

4. Register in main.go:
   ```go
   newModuleRepo := newmodule.NewRepository(db)
   newModuleService := newmodule.NewService(newModuleRepo, log)
   newModuleHandler := newmodule.NewHandler(newModuleService)
   
   group := api.Group("/newmodule")
   group.Use(authMiddleware(jwtManager))
   group.GET("", newModuleHandler.List)
   // ...
   ```

## Database Migrations

### Create New Migration

```bash
make migrate-create name=add_new_column
```

This creates:
- `migrations/NNNNNN_add_new_column.up.sql`
- `migrations/NNNNNN_add_new_column.down.sql`

### Run Migrations

```bash
# Up
make migrate-up

# Down (rollback last)
make migrate-down

# Specific version
migrate -path migrations -database "$DATABASE_URL" goto 3
```

### Migration Best Practices

1. **Always provide down migration**
2. **Use transactions** for multi-statement migrations
3. **Add indexes concurrently** in production:
   ```sql
   CREATE INDEX CONCURRENTLY idx_name ON table(column);
   ```
4. **Backfill data** in separate step
5. **Test migrations** locally before committing

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run with race detector
make test-race

# Run specific package
go test ./internal/auth/... -v

# Run specific test
go test ./internal/auth/... -run TestJWTManager_GenerateAndValidateAccessToken -v
```

### Integration Tests

```bash
# Requires running PostgreSQL and Redis
make test-integration

# Or run with testcontainers (if configured)
go test -tags=integration ./tests/...
```

### Test Patterns

**Table-driven tests:**
```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    Input
        expected Expected
        wantErr  bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"invalid input", invalidInput, nil, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

**Repository tests** use real database:
```go
func TestRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    repo := NewRepository(db)
    // test with real DB
}
```

## Code Style

### Formatting

```bash
# Format all code
make fmt

# Or manually
gofmt -w .
goimports -w .
```

### Linting

```bash
# Run linter
make lint

# Or manually
golangci-lint run ./...
```

### Common Patterns

**Error wrapping:**
```go
if err != nil {
    return fmt.Errorf("create user: %w", err)
}
```

**Context usage:**
```go
func (s *Service) DoSomething(ctx context.Context, id string) error {
    // Pass ctx to all downstream calls
    return s.repo.DoSomething(ctx, id)
}
```

**Structured logging:**
```go
log.Info().
    Str("user_id", userID).
    Str("action", "order_created").
    Float64("total", total).
    Msg("Order created")
```

## Debugging

### API Server

```bash
# Run with debug logging
LOG_LEVEL=debug go run ./cmd/api

# Or with delve
dlv debug ./cmd/api
```

### Database

```bash
# Connect to PostgreSQL
docker exec -it forgestock-postgres psql -U forgestock -d forgestock

# View migration status
migrate -path migrations -database "$DATABASE_URL" version
```

### Redis

```bash
# Connect to Redis
docker exec -it forgestock-redis redis-cli

# View keys
KEYS *

# Check queue
LRANGE notifications 0 -1
```

## Common Tasks

### Add New API Endpoint

1. Add route in `cmd/api/main.go`
2. Add handler method in `internal/<module>/handler.go`
3. Add service method in `internal/<module>/service.go`
4. Add repository method if needed
5. Add tests
6. Update API documentation

### Add New Database Column

1. Create migration: `make migrate-create name=add_column_to_table`
2. Edit `.up.sql` and `.down.sql`
3. Run migration: `make migrate-up`
4. Update repository queries
5. Update models if needed
6. Update tests

### Add New Configuration

1. Add field to `internal/platform/config/config.go`
2. Add default in `setDefaults()`
3. Use in code via `cfg.NewField`
4. Update `.env.example`
5. Update documentation

## Git Workflow

1. Create feature branch: `git checkout -b feature/new-feature`
2. Make changes with tests
3. Run checks: `make test lint fmt`
4. Commit with conventional commits:
   ```
   feat: add new endpoint for X
   
   - Add handler, service, repository
   - Add tests
   - Update documentation
   ```
5. Push and create PR
6. CI must pass
7. Code review
8. Merge to main

## Troubleshooting

### Database Connection Failed

```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Check logs
docker logs forgestock-postgres

# Verify connection string
echo $DATABASE_URL
```

### Migration Fails

```bash
# Check migration status
migrate -path migrations -database "$DATABASE_URL" version

# Force version (if dirty)
migrate -path migrations -database "$DATABASE_URL" force 1

# Check SQL syntax
psql $DATABASE_URL -f migrations/NNNNNN_name.up.sql
```

### Tests Fail

```bash
# Run with verbose output
go test ./... -v

# Run race detector
go test -race ./...

# Check test database exists
psql postgres://forgestock:forgestock@localhost:5432/forgestock_test?sslmode=disable -c "\dt"
```

### Port Already in Use

```bash
# Find process on port 8080
lsof -i :8080

# Kill it
kill -9 <PID>
```

## Useful Commands

```bash
# Build binaries
make build

# Clean build artifacts
make clean

# View all Makefile targets
make help

# Generate mocks (if using mockgen)
make generate

# View Docker logs
make docker-logs
```