# ForgeStock Architecture

## Overview

ForgeStock follows a clean, modular architecture with clear separation of concerns. The architecture is designed for maintainability, testability, and scalability.

## Architectural Principles

1. **Separation of Concerns**: Each layer has a single responsibility
2. **Dependency Inversion**: High-level modules don't depend on low-level modules
3. **Interface Segregation**: Small, focused interfaces
4. **Single Responsibility**: Each package/module has one reason to change
5. **Explicit Dependencies**: Dependencies are passed explicitly, not global

## Layer Structure

```
┌─────────────────────────────────────────────────────────────┐
│                     HTTP Handlers                             │
│  (Request/Response, Validation, Serialization)               │
├─────────────────────────────────────────────────────────────┤
│                      Services                                 │
│  (Business Logic, Use Cases, Orchestration)                  │
├─────────────────────────────────────────────────────────────┤
│                    Repositories                               │
│  (Data Access, SQL Queries, Transactions)                    │
├─────────────────────────────────────────────────────────────┤
│                   Domain Models                               │
│  (Entities, Value Objects, Domain Events)                    │
├─────────────────────────────────────────────────────────────┤
│                  Infrastructure                               │
│  (Database, Redis, HTTP Client, Logger, Metrics, Config)     │
└─────────────────────────────────────────────────────────────┘
```

## Module Organization

Each domain module follows this structure:

```
internal/<module>/
├── handler.go      # HTTP handlers
├── service.go      # Business logic
├── repository.go   # Data access
├── models.go       # Domain models (if separate)
└── *_test.go       # Tests
```

### Example: Orders Module

```
internal/orders/
├── handler.go      # Create, Get, List, UpdateStatus, Cancel
├── service.go      # Order creation, status transitions, inventory coordination
├── repository.go   # CRUD, status updates, listing with filters
├── models.go       # Order, OrderItem, Status constants
└── repository_test.go
```

## Dependency Flow

```
main.go
  ├── config.Load()
  ├── database.New()
  ├── redis.New()
  ├── logger.New()
  │
  ├── auth.NewService()
  │   ├── userRepo
  │   ├── jwtManager
  │   └── redis
  │
  ├── users.NewService()
  │   └── userRepo
  │
  ├── categories.NewService()
  │   └── categoryRepo
  │
  ├── products.NewService()
  │   ├── productRepo
  │   └── categoryRepo
  │
  ├── inventory.NewService()
  │   ├── inventoryRepo
  │   ├── productRepo
  │   └── db (for transactions)
  │
  ├── orders.NewService()
  │   ├── orderRepo
  │   ├── productRepo
  │   ├── inventoryRepo
  │   ├── inventoryService
  │   ├── paymentService
  │   └── db
  │
  └── payments.NewService()
      ├── paymentRepo
      ├── provider (interface)
      └── db
```

## Key Design Decisions

### 1. No Global State

All dependencies are explicitly passed through constructors. No `init()` functions with side effects.

### 2. Interfaces for External Dependencies

```go
// payments/provider.go
type Provider interface {
    Authorize(ctx context.Context, payment *Payment) error
    Capture(ctx context.Context, payment *Payment) error
    Refund(ctx context.Context, payment *Payment) error
    Void(ctx context.Context, payment *Payment) error
}
```

This allows:
- Easy testing with mocks
- Swapping implementations (Stripe, PayPal, etc.)
- Clear contract definition

### 3. Repository Pattern

Repositories encapsulate all SQL. Services don't know about database structure.

```go
type Repository interface {
    Create(ctx context.Context, order *Order) error
    GetByID(ctx context.Context, id string) (*Order, error)
    UpdateStatus(ctx context.Context, id, status string) error
    List(ctx context.Context, filter Filter, page, pageSize int) ([]*Order, int, error)
}
```

### 4. Database Transactions

Transactions are managed at the service layer for operations spanning multiple repositories.

```go
func (s *Service) Create(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    tx, err := s.db.BeginTx(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    // Create order and items in transaction
    if err := s.createOrderInTx(ctx, tx, order); err != nil {
        return nil, err
    }

    // Reserve inventory (separate transaction per item for optimistic locking)
    for _, item := range order.Items {
        s.inventorySvc.ReserveStock(...)
    }

    return tx.Commit(ctx)
}
```

### 5. Optimistic Locking for Inventory

Inventory uses a `version` column for optimistic locking:

```sql
UPDATE inventory 
SET available_qty = available_qty - $1, 
    reserved_qty = reserved_qty + $1,
    version = version + 1
WHERE id = $2 AND version = $3 AND available_qty >= $1
```

Retries with exponential backoff handle concurrent modifications.

### 6. Context Propagation

`context.Context` is passed through all layers for:
- Cancellation
- Timeouts
- Request-scoped values (request ID, user ID)

### 7. Structured Logging

All logging uses zerolog with structured fields:

```go
log.Info().
    Str("request_id", requestID).
    Str("user_id", userID).
    Str("action", "order_created").
    Float64("total", order.Total).
    Msg("Order created")
```

### 8. Error Handling

Typed errors with HTTP status codes:

```go
type AppError struct {
    Code       string
    Message    string
    StatusCode int
    Err        error
}

func NewNotFoundError(message string, err error) *AppError
func NewConflictError(message string, err error) *AppError
// etc.
```

### 9. Request/Response DTOs

Separate DTOs from domain models:

```go
// Request
type CreateOrderRequest struct {
    CustomerID string              `json:"customer_id" validate:"required"`
    Items      []CreateOrderItem   `json:"items" validate:"required,min=1,dive"`
    Currency   string              `json:"currency" validate:"required,len=3"`
}

// Response uses domain model directly
```

### 10. Middleware Chain

```
Request
  → RequestIDMiddleware
  → LoggerMiddleware
  → RecoveryMiddleware
  → CORSMiddleware
  → SecurityHeadersMiddleware
  → AuthMiddleware (if protected)
  → RequireRoles (if restricted)
  → Handler
```

## Background Worker

Separate process (`cmd/worker/main.go`) handles:
- Notification queue processing
- Low-stock alerts (periodic)
- Cleanup jobs (periodic)
- Audit/event processing

Uses Redis lists as simple job queue with `LPUSH`/`RPOP`.

## Security

- **Password Hashing**: bcrypt with default cost
- **JWT**: HS256, short-lived access tokens (15min), long-lived refresh tokens (7 days)
- **Refresh Token Rotation**: New refresh token issued on each refresh, old one revoked
- **Token Revocation**: Redis-based blacklist with TTL
- **Role-Based Access**: Middleware checks roles for protected routes
- **Rate Limiting**: Token bucket per IP (configurable)
- **Security Headers**: CSP, X-Frame-Options, etc.
- **CORS**: Configurable allowed origins

## Observability

### Metrics (Prometheus)

- `http_requests_total` - Counter by method, path, status
- `http_request_duration_seconds` - Histogram by method, path
- `database_operations_total` - Counter by operation, table, status
- `database_operation_duration_seconds` - Histogram by operation, table
- `background_jobs_total` - Counter by job_type, status
- `background_job_duration_seconds` - Histogram by job_type
- `cache_operations_total` - Counter by operation, status

### Health Checks

- `/health` - Liveness (process is running)
- `/ready` - Readiness (db + redis connections work)

### Logging

Structured JSON with:
- Request ID for correlation
- Timestamp (RFC3339)
- Level
- Message
- Structured fields

## Testing Strategy

### Unit Tests

- Test services with mocked repositories
- Test handlers with mocked services
- Table-driven tests for multiple scenarios
- `go test -race` for concurrency issues

### Integration Tests

- Real PostgreSQL and Redis (testcontainers or docker-compose)
- Test full request flow
- Test database constraints and transactions

### Test Organization

```
internal/<module>/
├── *_test.go           # Unit tests
tests/
├── integration/        # Integration tests
└── fixtures/           # Test data
```

## Migration Strategy

- Versioned SQL migrations in `migrations/`
- Up/down pairs for rollback
- Run on application startup or via Makefile
- Seed data separate migration

## Configuration Management

- `viper` for environment variable binding
- Strongly typed `Config` struct
- Defaults for development
- Required validation for production
- `.env.example` for documentation
- Never commit `.env`

## Future Considerations

1. **Event Sourcing**: Consider for audit-heavy domains
2. **CQRS**: Separate read/write models for complex queries
3. **Message Queue**: Replace Redis list with RabbitMQ/Kafka for reliability
4. **Distributed Tracing**: Add OpenTelemetry
5. **API Versioning**: URL versioning (`/api/v1/`) with deprecation policy