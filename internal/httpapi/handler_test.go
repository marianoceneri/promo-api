package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/promo-api/internal/domain"
	"example.com/promo-api/internal/httpapi"
	"example.com/promo-api/internal/repository"
	"example.com/promo-api/internal/service"
)

func TestCreateQuote(t *testing.T) {
	repo := repository.NewMemoryPromotionRepository()
	handler := httpapi.NewHandler(repo, service.NewQuoteService(repo))
	body := []byte(`{
		"at":"2026-08-10T10:00:00-03:00",
		"channel":"web",
		"customer":{"tier":"standard","first_purchase":false},
		"items":[{"sku":"PHONE","category":"electronics","quantity":1,"unit_price_cents":100000}],
		"coupon_codes":[]
	}`)

	request := httptest.NewRequest(http.MethodPost, "/v1/quotes", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	var quote domain.Quote
	if err := json.Unmarshal(response.Body.Bytes(), &quote); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if quote.TotalCents != 85000 {
		t.Fatalf("expected total 85000, got %d", quote.TotalCents)
	}
}

func TestCreateQuoteRejectsUnknownFields(t *testing.T) {
	repo := repository.NewMemoryPromotionRepository()
	handler := httpapi.NewHandler(repo, service.NewQuoteService(repo))
	request := httptest.NewRequest(http.MethodPost, "/v1/quotes", bytes.NewBufferString(`{"unexpected":true}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}
