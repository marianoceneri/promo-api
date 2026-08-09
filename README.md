# Promotions API

Small REST API that calculates promotions for a shopping cart. It is intentionally
kept independent from any contract or policy engine so it can be used as a
brownfield system in tooling experiments.

## Run

```bash
go run ./cmd/api
```

The server listens on `:8080` by default. Set `PORT` to change it.

## Endpoints

- `GET /health`
- `GET /v1/promotions`
- `POST /v1/quotes`

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
```
