# ForgeStock API Reference

## Base URL

```
Development: http://localhost:8080/api/v1
Production:  https://api.forgestock.com/api/v1
```

## Authentication

All endpoints (except `/auth/*` and `/health*`) require a Bearer token:

```
Authorization: Bearer <access_token>
```

### Token Types

| Token | TTL | Purpose |
|-------|-----|---------|
| Access Token | 15 minutes | API authorization |
| Refresh Token | 7 days | Obtain new access tokens |

### Refresh Token Rotation

- Each refresh generates a new refresh token
- Old refresh token is revoked
- Revoked tokens stored in Redis with TTL

## Error Responses

All errors follow this format:

```json
{
  "success": false,
  "error": {
    "code": "error_code",
    "message": "Human readable message",
    "details": "Additional details (optional)"
  },
  "request_id": "uuid"
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| validation_error | 400 | Invalid request payload |
| unauthorized | 401 | Missing or invalid token |
| forbidden | 403 | Insufficient permissions |
| not_found | 404 | Resource not found |
| conflict | 409 | Resource conflict (duplicate) |
| invalid_state_transition | 400 | Illegal status change |
| insufficient_inventory | 400 | Not enough stock |
| internal_error | 500 | Server error |

## Pagination

List endpoints support pagination:

```
GET /api/v1/products?page=1&page_size=20
```

Response includes meta:

```json
{
  "success": true,
  "data": [...],
  "request_id": "uuid",
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

## Filtering & Sorting

### Products

```
GET /api/v1/products?category_id=uuid&active=true&search=headphone&sort_by=price&sort_order=ASC
```

| Parameter | Type | Description |
|-----------|------|-------------|
| category_id | UUID | Filter by category |
| active | boolean | Filter by active status |
| search | string | Search name, SKU, description |
| sort_by | string | Field to sort (created_at, name, price) |
| sort_order | string | ASC or DESC |

### Inventory

```
GET /api/v1/inventory?product_id=uuid&warehouse_id=default&low_stock_only=true
```

### Orders

```
GET /api/v1/orders?customer_id=uuid&status=pending
```

## Endpoints

### Authentication

#### Register
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123",
  "first_name": "John",
  "last_name": "Doe",
  "role": "customer"
}
```

Response: 201 Created
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "request_id": "uuid"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

Response: 200 OK (same as register)

#### Refresh Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJ..."
}
```

Response: 200 OK (new token pair)

#### Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

Response: 200 OK
```json
{
  "success": true,
  "data": { "message": "Logged out successfully" },
  "request_id": "uuid"
}
```

#### Change Password
```http
POST /api/v1/auth/change-password
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "current_password": "oldpassword",
  "new_password": "newsecurepassword123"
}
```

Response: 200 OK

---

### Users

#### Get Profile
```http
GET /api/v1/users/me
Authorization: Bearer <access_token>
```

Response: 200 OK
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "customer",
    "active": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  },
  "request_id": "uuid"
}
```

#### Update Profile
```http
PATCH /api/v1/users/me
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "first_name": "Jane",
  "last_name": "Smith"
}
```

Response: 200 OK (updated user object)

---

#### List Users (Admin/Manager)
```http
GET /api/v1/users?page=1&page_size=20
Authorization: Bearer <access_token>
```

Response: 200 OK with pagination meta

#### Get User (Admin/Manager)
```http
GET /api/v1/users/:id
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Update User Role (Admin/Manager)
```http
PATCH /api/v1/users/:id/role
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "role": "manager"
}
```

Valid roles: `admin`, `manager`, `warehouse_operator`, `customer`

Response: 200 OK

#### Deactivate User (Admin/Manager)
```http
POST /api/v1/users/:id/deactivate
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Activate User (Admin/Manager)
```http
POST /api/v1/users/:id/activate
Authorization: Bearer <access_token>
```

Response: 200 OK

---

### Categories

#### List Categories
```http
GET /api/v1/categories?page=1&page_size=20
Authorization: Bearer <access_token>
```

Response: 200 OK with pagination

#### Get Category
```http
GET /api/v1/categories/:id
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Create Category (Admin/Manager)
```http
POST /api/v1/categories
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Electronics",
  "description": "Electronic devices"
}
```

Response: 201 Created

#### Update Category (Admin/Manager)
```http
PATCH /api/v1/categories/:id
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "name": "Updated Name",
  "description": "Updated description"
}
```

Response: 200 OK

#### Delete Category (Admin/Manager)
```http
DELETE /api/v1/categories/:id
Authorization: Bearer <access_token>
```

Response: 200 OK
```json
{
  "success": true,
  "data": { "message": "Category deleted successfully" },
  "request_id": "uuid"
}
```

---

### Products

#### List Products
```http
GET /api/v1/products?page=1&page_size=20&category_id=uuid&active=true&search=headphone
Authorization: Bearer <access_token>
```

Response: 200 OK with pagination

#### Get Product
```http
GET /api/v1/products/:id
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Create Product (Admin/Manager)
```http
POST /api/v1/products
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "sku": "ELEC-003",
  "name": "Bluetooth Speaker",
  "description": "Portable speaker",
  "category_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "price": 79.99,
  "cost": 40.00,
  "currency": "USD"
}
```

Response: 201 Created

#### Update Product (Admin/Manager)
```http
PATCH /api/v1/products/:id
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "price": 89.99,
  "active": false
}
```

Response: 200 OK

#### Delete Product (Admin/Manager)
```http
DELETE /api/v1/products/:id
Authorization: Bearer <access_token>
```

Response: 200 OK

---

### Inventory

#### List Inventory
```http
GET /api/v1/inventory?page=1&page_size=20&product_id=uuid&warehouse_id=default&low_stock_only=true
Authorization: Bearer <access_token>
```

Response: 200 OK with pagination

#### Get Inventory
```http
GET /api/v1/inventory/:id
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Get by Product & Warehouse
```http
GET /api/v1/inventory/by-product?product_id=uuid&warehouse_id=default
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Get Transactions
```http
GET /api/v1/inventory/:id/transactions?page=1&page_size=20
Authorization: Bearer <access_token>
```

Response: 200 OK with pagination

#### Reserve Stock
```http
POST /api/v1/inventory/reserve
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "product_id": "uuid",
  "warehouse_id": "default",
  "quantity": 5,
  "reference_id": "order-uuid",
  "reference_type": "order",
  "notes": "Reserved for order"
}
```

Response: 200 OK (updated inventory)

#### Release Stock
```http
POST /api/v1/inventory/release
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "product_id": "uuid",
  "warehouse_id": "default",
  "quantity": 5,
  "reference_id": "order-uuid",
  "reference_type": "order",
  "notes": "Released due to cancellation"
}
```

Response: 200 OK

#### Add Stock (Receive)
```http
POST /api/v1/inventory/add
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "product_id": "uuid",
  "warehouse_id": "default",
  "quantity": 100,
  "reference_id": "po-123",
  "reference_type": "purchase_order",
  "notes": "Received from supplier"
}
```

Response: 200 OK

#### Adjust Stock
```http
POST /api/v1/inventory/adjust
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "product_id": "uuid",
  "warehouse_id": "default",
  "available_delta": -5,
  "reserved_delta": 0,
  "damaged_delta": 2,
  "notes": "Damaged during handling"
}
```

Response: 200 OK

---

### Orders

#### Create Order
```http
POST /api/v1/orders
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "customer_id": "uuid",
  "items": [
    {
      "product_id": "uuid",
      "quantity": 2
    }
  ],
  "currency": "USD",
  "notes": "Gift wrapping requested"
}
```

Response: 201 Created
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "customer_id": "uuid",
    "status": "pending",
    "subtotal": 199.98,
    "tax": 20.00,
    "discount": 0.00,
    "total": 219.98,
    "currency": "USD",
    "notes": "Gift wrapping requested",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "items": [...]
  },
  "request_id": "uuid"
}
```

#### List Orders
```http
GET /api/v1/orders?page=1&page_size=20&customer_id=uuid&status=pending
Authorization: Bearer <access_token>
```

- Regular users see only their orders
- Admin/Manager see all orders

Response: 200 OK with pagination

#### Get Order
```http
GET /api/v1/orders/:id
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Cancel Order
```http
POST /api/v1/orders/:id/cancel
Authorization: Bearer <access_token>
```

Response: 200 OK
```json
{
  "success": true,
  "data": { "message": "Order cancelled successfully" },
  "request_id": "uuid"
}
```

#### Update Order Status (Admin/Manager)
```http
PATCH /api/v1/orders/:id/status
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "status": "confirmed"
}
```

Valid statuses: `pending`, `confirmed`, `processing`, `shipped`, `delivered`, `cancelled`

Response: 200 OK

---

### Payments

#### Authorize Payment
```http
POST /api/v1/payments/:id/authorize
Authorization: Bearer <access_token>
```

Response: 200 OK (updated payment)

#### Capture Payment
```http
POST /api/v1/payments/:id/capture
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Refund Payment
```http
POST /api/v1/payments/:id/refund
Authorization: Bearer <access_token>
```

Response: 200 OK

#### Void Payment
```http
POST /api/v1/payments/:id/void
Authorization: Bearer <access_token>
```

Response: 200 OK

---

### Health & Metrics

#### Health Check
```http
GET /api/v1/health
```

Response: 200 OK
```json
{
  "status": "healthy",
  "checks": { "database": "ok" }
}
```

#### Readiness Check
```http
GET /api/v1/ready
```

Response: 200 OK or 503 Service Unavailable
```json
{
  "status": "ready",
  "checks": {
    "database": "ready",
    "redis": "ready"
  }
}
```

#### Prometheus Metrics
```http
GET /metrics
```

Returns Prometheus format metrics.

## Rate Limiting

- Default: 100 requests/second per IP
- Configurable via `RATE_LIMIT_RPS`
- Returns 429 Too Many Requests when exceeded

## CORS

- Configurable via `CORS_ALLOWED_ORIGINS`
- Supports credentials
- Preflight handled automatically

## Request IDs

All responses include `X-Request-ID` header and `request_id` in JSON body for tracing.