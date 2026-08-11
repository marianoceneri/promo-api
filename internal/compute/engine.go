package compute

// This file is yours. LORD created it once and will never overwrite it.
// Implement the business logic the contract cannot express; the generated
// code around it enforces every contractual invariant on every response.

import (
	"fmt"
	"sort"

	"github.com/marianoceneri/promo-api/internal/domain"
)

type BusinessEngine struct{}

func NewBusinessEngine() *BusinessEngine { return &BusinessEngine{} }

// QuoteCart: el importe final de un carrito tras acumular las promociones publicadas, habilitadas y vigentes al instante de la cotización
//
// Contractual invariants (enforced on every response, and by derived tests):
//   - PR-DISCOUNT-NON-NEGATIVE: discount_cents is never negative
//   - PR-DISCOUNT-BOUNDED: discount_cents never exceeds subtotal_cents
//   - PR-TOTAL-CONSISTENT: total_cents equals subtotal_cents minus discount_cents
//   - PR-BREAKDOWN-ADDS-UP: discount_cents equals the sum of applied[].discount_cents
func (e *BusinessEngine) QuoteCart(request domain.QuoteRequest, promotions []domain.Promotion) (domain.Quote, error) {
	subtotal := int64(0)
	for _, item := range request.Items {
		if item.Quantity < 0 {
			return domain.Quote{}, fmt.Errorf("quantity must not be negative for sku %q", item.Sku)
		}
		if item.UnitPriceCents < 0 {
			return domain.Quote{}, fmt.Errorf("unit_price_cents must not be negative for sku %q", item.Sku)
		}
		subtotal += item.Quantity * item.UnitPriceCents
	}

	eligible := make([]domain.Promotion, 0, len(promotions))
	for _, promotion := range promotions {
		if isQuotable(promotion, request) {
			eligible = append(eligible, promotion)
		}
	}
	// The catalog order is not part of the contract; sorting by id keeps the
	// breakdown (and the capping below) deterministic across stores.
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].Id < eligible[j].Id })

	// Discounts accumulate over the subtotal. Each share is clamped to the
	// discount capacity still left, so the breakdown keeps adding up to
	// discount_cents even once the promotions exceed the cart.
	applied := make([]domain.AppliedPromotion, 0, len(eligible))
	discount := int64(0)
	for _, promotion := range eligible {
		share := subtotal * promotion.DiscountPercent / 100
		if remaining := subtotal - discount; share > remaining {
			share = remaining
		}
		if share < 0 {
			share = 0
		}
		discount += share
		applied = append(applied, domain.AppliedPromotion{PromotionId: promotion.Id, DiscountCents: share})
	}

	return domain.Quote{
		SubtotalCents: subtotal,
		DiscountCents: discount,
		TotalCents:    subtotal - discount,
		Applied:       applied,
	}, nil
}

// isQuotable mirrors the PromotionValidity policy (inclusive start, exclusive
// end) that CheckPromotionValidity already applies, restricted to promotions
// the catalog has published and left enabled.
func isQuotable(promotion domain.Promotion, request domain.QuoteRequest) bool {
	if promotion.Status != "published" || !promotion.Enabled {
		return false
	}
	return !request.At.Before(promotion.StartsAt) && request.At.Before(promotion.EndsAt)
}
