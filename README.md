# ForgeStock - Inventory and Order Management Platform

A production-grade Go backend for inventory and order management with clean architecture, comprehensive testing, and operational tooling.

## Features

- **User Management**: Registration, authentication, roles (admin, manager, warehouse_operator, customer)
- **Product Management**: CRUD, categories, search, filtering, pagination
- **Inventory Management**: Stock tracking, reservations, adjustments, optimistic locking
- **Order Management**: Full lifecycle (pending → confirmed → processing → shipped → delivered/cancelled)
- **Payment Processing**: Mock provider with authorization, capture, refund, void
- **Notifications**: Email abstraction with background processing
- **Audit Logging**: Comprehensive audit trail for all critical operations
- **Observability**: Structured JSON logging, Prometheus metrics, health/readiness checks
- **Security**: JWT authentication, bcrypt password hashing, role-based authorization, rate limiting

## Architecture

```
forge-stock/
├── cmd/
│   ├── api/           # HTTP API server
│   └── worker/        # Background job processor
├── internal/
│   ├── auth/          # Authentication & authorization
│   ├── users/         # User management
│   ├── products/      # Product management
│   ├── categories/    # Category management
│   ├── inventory/     # Inventory management
│   ├── orders/        # Order management
│   ├── payments/      # Payment processing
│   ├── notifications/ # Notification system
│   ├── audit/         # Audit logging
│   ├── health/        # Health checks
│   └── platform/      # Shared infrastructure
│       ├── database/  # PostgreSQL connection pool
│       ├── redis/     # Redis client
│       ├── http/      # HTTP middleware, responses, errors
│       ├── logger/    # Structured logging
│       ├── metrics/   # Prometheus metrics
│       └── config/    # Configuration management
├── migrations/        # Database migrations
├── tests/             # Integration tests
├── scripts/           # Operational scripts
├── deployments/       # Docker, Kubernetes configs
├── docs/              # Documentation
└── .github/workflows/ # CI/CD pipelines
```

## Technology Stack

- **Language**: Go 1.24+
- **Database**: PostgreSQL 16+
- **Cache/Queue**: Redis 7+
- **Framework**: Gin HTTP framework
- **Migrations**: golang-migrate
- **Auth**: JWT with refresh token rotation
- **Password Hashing**: bcrypt
- **Logging**: zerolog (structured JSON)
- **Metrics**: Prometheus client
- **Validation**: go-playground/validator
- **Testing**: testify, table-driven tests
- **Containerization**: Docker, Docker Compose
- **CI/CD**: GitHub Actions

## Quick Start

### Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Redis 7+
- golang-migrate (for migrations)

### Local Development

1. **Clone and setup**
   ```bash
   git clone <repo>
   cd forgestock
   cp .env.example .env
   ```

2. **Start dependencies**
   ```bash
   make docker-up
   ```

3. **Run migrations**
   ```bash
   make migrate-up
   ```

4. **Seed development data**
   ```bash
   make seed
   ```

5. **Run API server**
   ```bash
   make run
   ```

6. **Run worker (separate terminal)**
   ```bash
   make run-worker
   ```

### Using Docker Compose

```bash
docker-compose up -d
```

The API will be available at `http://localhost:8080`

## API Endpoints

### Authentication
```
POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/refresh
POST   /api/v1/auth/logout
POST   /api/v1/auth/change-password
```

### Users
```
GET    /api/v1/users/me
PATCH  /api/v1/users/me
GET    /api/v1/users              # admin/manager
GET    /api/v1/users/:id          # admin/manager
PATCH  /api/v1/users/:id/role     # admin/manager
POST   /api/v1/users/:id/deactivate # admin/manager
POST   /api/v1/users/:id/activate   # admin/manager
```

### Categories
```
GET    /api/v1/categories
GET    /api/v1/categories/:id
POST   /api/v1/categories          # admin/manager
PATCH  /api/v1/categories/:id      # admin/manager
DELETE /api/v1/categories/:id      # admin/manager
```

### Products
```
GET    /api/v1/products
GET    /api/v1/products/:id
POST   /api/v1/products            # admin/manager
PATCH  /api/v1/products/:id        # admin/manager
DELETE /api/v1/products/:id        # admin/manager
```

### Inventory
```
GET    /api/v1/inventory
GET    /api/v1/inventory/:id
GET    /api/v1/inventory/by-product
GET    /api/v1/inventory/:id/transactions
POST   /api/v1/inventory/reserve
POST   /api/v1/inventory/release
POST   /api/v1/inventory/add
POST   /api/v1/inventory/adjust
```

### Orders
```
POST   /api/v1/orders
GET    /api/v1/orders
GET    /api/v1/orders/:id
POST   /api/v1/orders/:id/cancel
PATCH  /api/v1/orders/:id/status   # admin/manager
```

### Payments
```
POST   /api/v1/payments/:id/authorize
POST   /api/v1/payments/:id/capture
POST   /api/v1/payments/:id/refund
POST   /api/v1/payments/:id/void
```

### Health & Metrics
```
GET    /api/v1/health
GET    /api/v1/ready
GET    /metrics
```

## Authentication

All endpoints (except auth) require a Bearer token:

```bash
curl -H "Authorization: Bearer <access_token>" http://localhost:8080/api/v1/users/me
```

### Register
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123","first_name":"John","last_name":"Doe"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

## Default Users (after seeding)

| Email | Password | Role |
|-------|----------|------|
| admin@forgestock.com | password123 | admin |
| manager@forgestock.com | password123 | manager |
| operator@forgestock.com | password123 | warehouse_operator |
| customer@forgestock.com | password123 | customer |

## Testing

```bash
# Unit tests
make test

# Race detector
make test-race

# Integration tests (requires running Postgres/Redis)
make test-integration

# All tests with coverage
go test -race -coverprofile=coverage.out ./...
```

## Configuration

Environment variables (see `.env.example`):

| Variable | Description | Default |
|----------|-------------|---------|
| APP_ENV | Environment (development/production) | development |
| HTTP_PORT | HTTP server port | 8080 |
| DATABASE_URL | PostgreSQL connection string | postgres://forgestock:forgestock@localhost:5432/forgestock?sslmode=disable |
| REDIS_URL | Redis connection string | redis://localhost:6379/0 |
| JWT_SECRET | JWT signing secret | (required) |
| ACCESS_TOKEN_TTL | Access token TTL | 15m |
| REFRESH_TOKEN_TTL | Refresh token TTL | 168h |
| LOG_LEVEL | Log level (debug/info/warn/error) | info |
| RATE_LIMIT_ENABLED | Enable rate limiting | true |
| RATE_LIMIT_RPS | Requests per second | 100 |

## Database Migrations

```bash
# Create new migration
make migrate-create name=add_new_table

# Run migrations up
make migrate-up

# Run migrations down
make migrate-down
```

## Project Structure Details

### Clean Architecture Layers

1. **Handlers** (`internal/*/handler.go`) - HTTP request/response handling, validation
2. **Services** (`internal/*/service.go`) - Business logic, use cases
3. **Repositories** (`internal/*/repository.go`) - Data access, SQL queries
4. **Models** - Domain entities
4. **Platform** - Shared infrastructure (db, redis, http, logger, metrics, config)

### Error Handling

Typed application errors:
- `validation_error` - Invalid input
- `unauthorized` - Authentication required
- `forbidden` - Insufficient permissions
- `not_found` - Resource not found
- `conflict` - Resource conflict (duplicate)
- `invalid_state_transition` - Illegal state change
- `insufficient_inventory` - Not enough stock
- `internal_error` - Server error

### Concurrency Control

Inventory uses optimistic locking with version column to prevent race conditions during concurrent stock operations.

## Deployment

### Docker

```bash
# Build images
make docker-build

# Run with Docker Compose
docker-compose up -d
```

### Kubernetes

See `deployments/k8s/` for Kubernetes manifests.

## Documentation

- [Architecture](docs/architecture.md)
- [Database Schema](docs/database.md)
- [API Reference](docs/api.md)
- [Development Guide](docs/development.md)
- [Operations](docs/operations.md)

## License

MIT License