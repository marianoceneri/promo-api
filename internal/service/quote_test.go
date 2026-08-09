package service_test

import (
	"context"
	"testing"
	"time"

	"example.com/promo-api/internal/domain"
	"example.com/promo-api/internal/repository"
	"example.com/promo-api/internal/service"
)

func TestWeekdayPromotionAppliesWithCap(t *testing.T) {
	quote := mustQuote(t, requestAt("2026-08-10T10:00:00-03:00", "web", false, "standard",
		item("PHONE", "electronics", 1, 100000)))

	if quote.DiscountCents != 15000 {
		t.Fatalf("expected capped discount 15000, got %d", quote.DiscountCents)
	}
	if len(quote.Applied) != 1 || quote.Applied[0].ID != "WEEKDAY20" {
		t.Fatalf("expected WEEKDAY20, got %#v", quote.Applied)
	}
}

func TestWeekdayPromotionIsRejectedOutsideDailyWindow(t *testing.T) {
	quote := mustQuote(t, requestAt("2026-08-10T18:00:00-03:00", "web", false, "standard",
		item("PHONE", "electronics", 1, 100000)))

	if quote.DiscountCents != 0 {
		t.Fatalf("expected no discount, got %d", quote.DiscountCents)
	}
	assertReason(t, quote, "WEEKDAY20", "outside_daily_window")
}

func TestOvernightWindowIncludesTimeAfterMidnight(t *testing.T) {
	quote := mustQuote(t, requestAt("2026-08-11T01:30:00-03:00", "app", true, "standard",
		item("HEADPHONES", "electronics", 1, 30000)))

	if quote.DiscountCents != 3000 {
		t.Fatalf("expected 3000 discount, got %d", quote.DiscountCents)
	}
	if len(quote.Applied) != 1 || quote.Applied[0].ID != "APP_NIGHT10" {
		t.Fatalf("expected APP_NIGHT10, got %#v", quote.Applied)
	}
}

func TestStackablePromotionsCanBeatExclusivePromotion(t *testing.T) {
	request := requestAt("2026-08-22T23:00:00-03:00", "app", true, "vip",
		item("BOOK-A", "books", 2, 30000))
	request.CouponCodes = []string{"BACK2SCHOOL"}
	quote := mustQuote(t, request)

	if quote.DiscountCents != 22000 {
		t.Fatalf("expected 22000 combined discount, got %d", quote.DiscountCents)
	}
	if len(quote.Applied) != 3 {
		t.Fatalf("expected three stackable promotions, got %#v", quote.Applied)
	}
}

func TestCouponPromotionOnlyDiscountsApplicableCategory(t *testing.T) {
	request := requestAt("2026-08-18T20:00:00-03:00", "web", false, "standard",
		item("BOOK-A", "books", 1, 20000),
		item("MOUSE", "electronics", 1, 20000))
	request.CouponCodes = []string{"BACK2SCHOOL"}
	quote := mustQuote(t, request)

	if quote.DiscountCents != 5000 {
		t.Fatalf("expected books-only discount 5000, got %d", quote.DiscountCents)
	}
}

func requestAt(at, channel string, firstPurchase bool, tier string, items ...domain.CartItem) domain.QuoteRequest {
	return domain.QuoteRequest{
		At: mustTime(at), Channel: channel, Customer: domain.Customer{Tier: tier, FirstPurchase: firstPurchase}, Items: items,
	}
}

func item(sku, category string, quantity int, unitPrice int64) domain.CartItem {
	return domain.CartItem{SKU: sku, Category: category, Quantity: quantity, UnitPriceCents: unitPrice}
}

func mustQuote(t *testing.T, request domain.QuoteRequest) domain.Quote {
	t.Helper()
	repo := repository.NewMemoryPromotionRepository()
	quote, err := service.NewQuoteService(repo).Quote(context.Background(), request)
	if err != nil {
		t.Fatalf("quote failed: %v", err)
	}
	return quote
}

func mustTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func assertReason(t *testing.T, quote domain.Quote, promotionID, expected string) {
	t.Helper()
	for _, evaluation := range quote.Evaluations {
		if evaluation.ID != promotionID {
			continue
		}
		for _, reason := range evaluation.RejectionReasons {
			if reason == expected {
				return
			}
		}
		t.Fatalf("expected reason %q in %#v", expected, evaluation.RejectionReasons)
	}
	t.Fatalf("promotion %s was not evaluated", promotionID)
}
