# Promotions API

Small REST API written in Go that evaluates promotions for a shopping cart. It
is the independent brownfield system used to exercise
[LORD](https://github.com/marianoceneri/lord): LORD must discover its operational
rules from the existing code and tests before proposing changes.

This repository intentionally contains no LORD contract and has no runtime
dependency on LORD.

## Run

```bash
go run ./cmd/api
```

Requirements: Go 1.24 or newer.

The server listens on `:8080` by default. Set `PORT` to change it:

```bash
PORT=9090 go run ./cmd/api
```

## Endpoints

- `GET /health` returns service health.
- `GET /v1/promotions` lists the seeded promotion definitions.
- `POST /v1/quotes` evaluates every promotion and returns the selected discounts
  together with rejection reasons for non-eligible promotions.

Example quote:

```bash
curl -s http://localhost:8080/v1/quotes \
  -H 'content-type: application/json' \
  -d '{
    "at": "2026-08-10T10:00:00-03:00",
    "channel": "web",
    "customer": {"tier": "standard", "first_purchase": false},
    "items": [
      {"sku": "PHONE-1", "category": "electronics", "quantity": 1, "unit_price_cents": 80000}
    ],
    "coupon_codes": []
  }'
```

## Test

```bash
go test ./...
go vet ./...
```

The automated suite covers date and daily-time windows, overnight windows,
discount caps, coupon/category restrictions, exclusive versus stackable
selection, and HTTP input validation.

## Structure

```text
cmd/api                 HTTP server and graceful shutdown
internal/domain         Domain data structures
internal/repository     In-memory promotion catalog
internal/service        Eligibility and discount-selection logic
internal/httpapi        REST transport and request validation
```

## Scope

This is a deterministic engineering fixture, not a production commerce API. It
uses fixed August 2026 seed data and in-memory storage. Authentication,
persistence, administrative mutation, observability and distributed-system
concerns are deliberately outside its current scope.
