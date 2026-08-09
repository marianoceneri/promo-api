package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marianoceneri/promo-api/internal/domain"
	"github.com/marianoceneri/promo-api/internal/httpapi"
	"github.com/marianoceneri/promo-api/internal/repository"
	"github.com/marianoceneri/promo-api/internal/service"
)

func TestCreateQuote(t *testing.T) {
	repo := repository.NewMemoryPromotionRepository()
	handler := newTestHandler(repo)
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
	handler := newTestHandler(repo)
	request := httptest.NewRequest(http.MethodPost, "/v1/quotes", bytes.NewBufferString(`{"unexpected":true}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestCreatePromotionMakesItImmediatelyAvailable(t *testing.T) {
	repo := repository.NewMemoryPromotionRepositoryWith(nil)
	handler := newTestHandler(repo)
	body := []byte(`{
		"id":"FLASH15",
		"name":"Flash 15%",
		"starts_at":"2026-08-01T00:00:00-03:00",
		"ends_at":"2026-09-01T00:00:00-03:00",
		"timezone":"America/Argentina/Buenos_Aires",
		"daily_start":"00:00",
		"daily_end":"00:00",
		"weekdays":[0,1,2,3,4,5,6],
		"minimum_subtotal_cents":10000,
		"percent_off":15,
		"maximum_discount_cents":5000,
		"channels":["web"],
		"stackable":true,
		"priority":10
	}`)

	create := httptest.NewRequest(http.MethodPost, "/v1/promotions", bytes.NewReader(body))
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", created.Code, created.Body.String())
	}

	quote := httptest.NewRequest(http.MethodPost, "/v1/quotes", bytes.NewBufferString(`{
		"at":"2026-08-10T10:00:00-03:00",
		"channel":"web",
		"customer":{"tier":"standard","first_purchase":false},
		"items":[{"sku":"BOOK","category":"books","quantity":1,"unit_price_cents":20000}],
		"coupon_codes":[]
	}`))
	quoted := httptest.NewRecorder()
	handler.ServeHTTP(quoted, quote)
	if quoted.Code != http.StatusOK {
		t.Fatalf("expected quote status 200, got %d: %s", quoted.Code, quoted.Body.String())
	}
	var result domain.Quote
	if err := json.Unmarshal(quoted.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode quote: %v", err)
	}
	if result.DiscountCents != 3000 {
		t.Fatalf("expected new promotion discount 3000, got %d", result.DiscountCents)
	}
}

func TestCreatePromotionRejectsDuplicateID(t *testing.T) {
	repo := repository.NewMemoryPromotionRepository()
	handler := newTestHandler(repo)
	body := []byte(`{
		"id":"WEEKDAY20","name":"Duplicate",
		"starts_at":"2026-08-01T00:00:00-03:00","ends_at":"2026-09-01T00:00:00-03:00",
		"timezone":"UTC","daily_start":"00:00","daily_end":"00:00",
		"weekdays":[1],"percent_off":10,"channels":["web"]
	}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/promotions", bytes.NewReader(body)))
	if response.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", response.Code, response.Body.String())
	}
}

func TestCreatePromotionRejectsInvalidDefinition(t *testing.T) {
	repo := repository.NewMemoryPromotionRepositoryWith(nil)
	handler := newTestHandler(repo)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/promotions", bytes.NewBufferString(`{"id":"BROKEN"}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body.String())
	}
}

func newTestHandler(repo repository.PromotionRepository) http.Handler {
	return httpapi.NewHandler(repo, service.NewPromotionService(repo), service.NewQuoteService(repo))
}
