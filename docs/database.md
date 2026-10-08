# ForgeStock Database Schema

## Overview

PostgreSQL 16+ with normalized schema, proper constraints, indexes, and migrations.

## Tables

### users

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| email | VARCHAR(255) | UNIQUE NOT NULL |
| password | VARCHAR(255) | NOT NULL |
| first_name | VARCHAR(100) | NOT NULL |
| last_name | VARCHAR(100) | NOT NULL |
| role | VARCHAR(50) | NOT NULL DEFAULT 'customer' |
| active | BOOLEAN | NOT NULL DEFAULT true |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Indexes**: `idx_users_email`, `idx_users_role`, `idx_users_active`

### categories

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| name | VARCHAR(100) | UNIQUE NOT NULL |
| description | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Indexes**: `idx_categories_name`

### products

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| sku | VARCHAR(50) | UNIQUE NOT NULL |
| name | VARCHAR(200) | NOT NULL |
| description | TEXT | |
| category_id | UUID | NOT NULL REFERENCES categories(id) |
| price | DECIMAL(12,2) | NOT NULL CHECK (price >= 0) |
| cost | DECIMAL(12,2) | NOT NULL CHECK (cost >= 0) |
| currency | CHAR(3) | NOT NULL DEFAULT 'USD' |
| active | BOOLEAN | NOT NULL DEFAULT true |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Indexes**: `idx_products_sku`, `idx_products_category`, `idx_products_active`, `idx_products_name`

### inventory

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| product_id | UUID | NOT NULL REFERENCES products(id) |
| warehouse_id | VARCHAR(50) | NOT NULL |
| available_qty | INTEGER | NOT NULL DEFAULT 0 CHECK (available_qty >= 0) |
| reserved_qty | INTEGER | NOT NULL DEFAULT 0 CHECK (reserved_qty >= 0) |
| damaged_qty | INTEGER | NOT NULL DEFAULT 0 CHECK (damaged_qty >= 0) |
| reorder_threshold | INTEGER | NOT NULL DEFAULT 0 CHECK (reorder_threshold >= 0) |
| version | INTEGER | NOT NULL DEFAULT 1 |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Constraints**: UNIQUE(product_id, warehouse_id)

**Indexes**: `idx_inventory_product`, `idx_inventory_warehouse`, `idx_inventory_low_stock` (partial)

### inventory_transactions

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| inventory_id | UUID | NOT NULL REFERENCES inventory(id) |
| type | VARCHAR(50) | NOT NULL |
| quantity | INTEGER | NOT NULL |
| reference_id | VARCHAR(100) | |
| reference_type | VARCHAR(50) | |
| notes | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Indexes**: `idx_inventory_transactions_inventory`, `idx_inventory_transactions_reference`, `idx_inventory_transactions_created`

### orders

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| customer_id | UUID | NOT NULL REFERENCES users(id) |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' |
| subtotal | DECIMAL(12,2) | NOT NULL CHECK (subtotal >= 0) |
| tax | DECIMAL(12,2) | NOT NULL CHECK (tax >= 0) |
| discount | DECIMAL(12,2) | NOT NULL DEFAULT 0 CHECK (discount >= 0) |
| total | DECIMAL(12,2) | NOT NULL CHECK (total >= 0) |
| currency | CHAR(3) | NOT NULL DEFAULT 'USD' |
| notes | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Indexes**: `idx_orders_customer`, `idx_orders_status`, `idx_orders_created`

### order_items

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| order_id | UUID | NOT NULL REFERENCES orders(id) ON DELETE CASCADE |
| product_id | UUID | NOT NULL REFERENCES products(id) |
| sku | VARCHAR(50) | NOT NULL |
| name | VARCHAR(200) | NOT NULL |
| quantity | INTEGER | NOT NULL CHECK (quantity > 0) |
| price | DECIMAL(12,2) | NOT NULL CHECK (price >= 0) |
| total | DECIMAL(12,2) | NOT NULL CHECK (total >= 0) |

**Indexes**: `idx_order_items_order`, `idx_order_items_product`

### payments

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| order_id | UUID | NOT NULL REFERENCES orders(id) |
| amount | DECIMAL(12,2) | NOT NULL CHECK (amount >= 0) |
| currency | CHAR(3) | NOT NULL DEFAULT 'USD' |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' |
| provider | VARCHAR(50) | NOT NULL DEFAULT 'mock' |
| provider_ref | VARCHAR(100) | |
| failure_reason | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Indexes**: `idx_payments_order`, `idx_payments_status`

### notifications

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| type | VARCHAR(50) | NOT NULL |
| recipient | VARCHAR(255) | NOT NULL |
| subject | VARCHAR(255) | NOT NULL |
| body | TEXT | NOT NULL |
| status | VARCHAR(50) | NOT NULL DEFAULT 'pending' |
| metadata | JSONB | |
| error | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |
| sent_at | TIMESTAMPTZ | |

**Indexes**: `idx_notifications_status`, `idx_notifications_created`

### audit_logs

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PRIMARY KEY |
| actor_id | UUID | |
| actor_type | VARCHAR(50) | |
| action | VARCHAR(100) | NOT NULL |
| entity_type | VARCHAR(50) | NOT NULL |
| entity_id | UUID | NOT NULL |
| metadata | JSONB | |
| ip_address | INET | |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT NOW() |

**Indexes**: `idx_audit_logs_actor`, `idx_audit_logs_entity`, `idx_audit_logs_action`, `idx_audit_logs_created`

## Order Status Flow

```
PENDING → CONFIRMED → PROCESSING → SHIPPED → DELIVERED
    ↓         ↓           ↓           ↓
  CANCELLED  CANCELLED  CANCELLED  CANCELLED
```

Valid transitions enforced in application code.

## Payment Status Flow

```
PENDING → AUTHORIZED → CAPTURED → REFUNDED
    ↓         ↓
  FAILED    VOIDED
```

## Inventory Transaction Types

- `initial` - Initial inventory creation
- `receive` - Stock received (purchase order)
- `reserve` - Stock reserved for order
- `release` - Reservation released (cancelled order)
- `deduct` - Stock deducted (order shipped)
- `adjustment` - Manual adjustment
- `damage` - Damaged stock recorded

## Migration Management

Migrations are in `migrations/` with format:
- `NNNNNN_description.up.sql` - Forward migration
- `NNNNNN_description.down.sql` - Rollback migration

Run with:
```bash
# Using migrate CLI
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" down

# Using Makefile
make migrate-up
make migrate-down
```

## Seed Data

Development seed data in `migrations/000002_seed_data.up.sql`:
- 4 users (admin, manager, warehouse_operator, customer)
- 4 categories
- 6 products
- 6 inventory records
- 3 orders with items
- 3 payments

Password for all seeded users: `password123`

Run seed:
```bash
make seed
# or
migrate -path migrations -database "$DATABASE_URL" up 2
```

## Performance Considerations

1. **Indexes**: Created for all foreign keys and frequently queried columns
2. **Partial Index**: `idx_inventory_low_stock` only indexes rows needing reorder
3. **Check Constraints**: Prevent negative quantities at database level
4. **Foreign Keys**: Cascade delete for order_items when order deleted
5. **JSONB**: Used for flexible metadata in notifications and audit_logs

## Connection Pooling

Configured in `internal/platform/database/database.go`:
- MaxConns: 25
- MinConns: 5
- MaxConnLifetime: 1 hour
- MaxConnIdleTime: 30 minutes
- HealthCheckPeriod: 1 minute