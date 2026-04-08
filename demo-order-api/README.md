# Demo Order API

Go order processing service for the automated incident response demo. Handles order creation, retrieval, listing, and refunds, with an embedded web UI.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | Embedded frontend UI |
| `GET` | `/health` | Health check (ALB target group) |
| `POST` | `/orders` | Create order with optional discount code |
| `GET` | `/orders/{id}` | Get order by ID |
| `GET` | `/orders` | List orders (optional `?status=` filter) |
| `POST` | `/orders/{id}/refund` | Process refund |

## Request/Response Examples

### Create Order

```bash
curl -X POST https://$APP_DOMAIN/orders \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"name": "Widget", "quantity": 2, "unit_price": 29.99},
      {"name": "Gadget", "quantity": 1, "unit_price": 49.99}
    ],
    "discount_code": "SAVE10"
  }'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "items": [...],
  "subtotal": 109.97,
  "discount_code": "SAVE10",
  "discount_amount": 10.997,
  "total": 98.973,
  "status": "confirmed",
  "created_at": "2026-03-13T10:00:00Z"
}
```

### Get Order

```bash
curl https://$APP_DOMAIN/orders/{id}
```

### List Orders

```bash
curl https://$APP_DOMAIN/orders?status=confirmed
```

### Refund Order

```bash
curl -X POST https://$APP_DOMAIN/orders/{id}/refund
```

## Discount Codes

| Code | Tier | Discount |
|------|------|----------|
| `SAVE5` | Bronze | 5% |
| `SAVE10` | Silver | 10% |
| `SAVE15` | Gold | 15% |

The API also contains a deliberate bug triggered by a specific discount code. This is the demo trigger — the triage agent discovers and diagnoses it autonomously.

## DynamoDB Schema

- **Table**: `demo-orders`
- **Partition key**: `id` (String, UUID)
- **GSI**: `status-index` — partition key `status` (for list-by-status queries)

### Item Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | String | UUID, auto-generated |
| `items` | List | Order line items |
| `subtotal` | Number | Sum of item totals before discount |
| `discount_code` | String | Applied discount code (optional) |
| `discount_amount` | Number | Discount value applied |
| `total` | Number | Final total after discount |
| `status` | String | `pending`, `confirmed`, or `refunded` |
| `created_at` | String | ISO 8601 timestamp |
| `updated_at` | String | ISO 8601 timestamp |

## Observability

### Structured Logging

JSON via `slog` with fields: `timestamp`, `level`, `msg`, `request_id`, `method`, `path`, `status_code`, `duration_ms`, `error`, `trace_id`.

### CloudWatch Custom Metrics

Namespace: `DemoOrderAPI`

| Metric | Dimensions |
|--------|------------|
| `RequestCount` | Endpoint, Method, StatusCode |
| `ErrorCount` | Endpoint, Method, StatusCode |
| `Latency` | Endpoint, Method, StatusCode |

### X-Ray

Full tracing on all HTTP requests and DynamoDB calls. Trace ID propagated to structured logs.

### Recovery Middleware

Catches panics, logs full stack trace as structured JSON, returns HTTP 500. Critical for the demo — the agent reads these stack traces.

## Project Structure

```
├── cmd/api/main.go          # Entry point, server setup, route registration
├── internal/
│   ├── handler/             # HTTP handlers
│   │   ├── orders.go        # Create, get, list, refund
│   │   └── health.go        # Health check
│   ├── discount/            # Discount tier logic (BUG lives here)
│   │   └── discount.go
│   ├── model/               # Order, Item, DiscountTier structs
│   ├── store/               # DynamoDB operations
│   ├── middleware/           # Logging, metrics, X-Ray, recovery
│   └── observability/       # CloudWatch metrics helpers, X-Ray setup
├── web/                     # Embedded frontend UI
│   ├── static/
│   │   ├── index.html
│   │   ├── app.js           # Alpine.js or vanilla JS
│   │   └── style.css
│   └── embed.go             # go:embed directives
├── Dockerfile               # Multi-stage build
├── go.mod
└── go.sum
```

## Build

```bash
# Local development
go run ./cmd/api

# Docker build
docker build -t demo-order-api .

# Build and push to ECR (from parent repo)
just build-app
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `DYNAMODB_TABLE` | DynamoDB table name | `demo-orders` |
| `AWS_REGION` | AWS region | `eu-west-2` |

## Testing

```bash
go test ./...
```

Tests cover valid discount codes, handler happy paths, and store marshalling.
