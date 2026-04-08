# Demo Order API

Go order processing service for the automated incident response demo. Runs on ECS Fargate behind an ALB. The domain is configured via `.env` at the project root.

## API Endpoints

### Orders domain (`orders.{subdomain}`)
- `GET /` — Embedded frontend UI
- `GET /health` — Health check (ALB target group)
- `POST /orders` — Create order with optional discount code
- `GET /orders/{id}` — Get order by ID
- `GET /orders` — List orders (optional `?status=` filter)
- `POST /orders/{id}/refund` — Process refund

### Observer domain (`observer.{subdomain}`)
- `GET /` — Observer SPA (embedded static files)
- `GET /health` — Health check
- `GET /config` — Returns `{region, repo}` from environment variables
- `GET /api/agent-events/*` — Agent event streaming (shared with orders domain)

Host-based routing in `cmd/api/main.go` dispatches to the correct router based on the `Host` header.

## The Intentional Bug

In `internal/discount/discount.go`, `WELCOME` maps to tier index 3 but only 3 tiers exist (indices 0–2). This causes `panic: index out of range [3] with length 3`. **Do not fix this** — it is the demo trigger.

## Domain Models

- **Order**: id (UUID), items (list), subtotal, discount_code, discount_amount, total, status (pending/confirmed/refunded), created_at, updated_at
- **Item**: name, quantity, unit_price
- **DiscountTier**: name (bronze/silver/gold), rate (0.05/0.10/0.15)
- **Discount codes**: SAVE5 → bronze (5%), SAVE10 → silver (10%), SAVE15 → gold (15%), WELCOME → BUG (index 3, missing tier)

## DynamoDB Schema

- Table: `demo-orders`
- Partition key: `id` (string, UUID)
- GSI: `status-index` on `status` (for list-by-status queries)

## Observability

- Structured JSON logging via `slog` (fields: `request_id`, `trace_id`, `method`, `path`, `status_code`, `duration_ms`, `error`)
- CloudWatch metrics: `LogRequestCount` and `LogErrorCount` are derived from log metric filters (not published by the app). Per-request custom metrics (`RequestCount`, `ErrorCount`, `Latency`) were removed to avoid high-cardinality dimension explosion
- AWS X-Ray tracing on all requests and DynamoDB calls
- Recovery middleware catches panics and logs full stack traces

## Project Structure

```
demo-order-api/
├── cmd/api/main.go          # Entry point, server setup, route registration
├── internal/
│   ├── handler/             # HTTP handlers
│   │   ├── orders.go        # Create, get, list, refund
│   │   └── health.go        # Health check
│   ├── discount/            # Discount tier logic (BUG lives here)
│   │   └── discount.go
│   ├── model/               # Order, Item, DiscountTier structs
│   ├── store/               # DynamoDB operations
│   ├── middleware/           # Logging, X-Ray, recovery
│   └── observability/       # X-Ray setup
├── observer/                # Embedded observer UI (static/ populated at Docker build)
│   └── embed.go             # go:embed directives
├── web/                     # Embedded frontend UI
│   ├── static/
│   │   ├── index.html
│   │   ├── app.js
│   │   └── style.css
│   └── embed.go             # go:embed directives
├── Dockerfile
├── go.mod
└── go.sum
```

## Build

Multi-stage Docker build. Frontend is embedded via `go:embed` from `web/` directory.

## Conventions

- British English in all comments, logs, and UI text
- Standard library + `chi` router + `aws-sdk-go-v2`
- Table-driven tests
- Tests deliberately omit the WELCOME discount code
