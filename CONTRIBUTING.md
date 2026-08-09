# Contributing

This repository is both a Go service and a brownfield fixture for LORD. Changes
should remain realistic from an application-development perspective; do not add
generated LORD contracts or make the service depend on LORD at runtime.

Before committing:

```bash
gofmt -w .
go vet ./...
go test -race ./...
```

Add tests for business-rule changes, especially boundary times, promotion
selection, category scopes and invalid HTTP input.
