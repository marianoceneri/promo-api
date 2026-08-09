package domain

import "time"

type Promotion struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	StartsAt             time.Time      `json:"starts_at"`
	EndsAt               time.Time      `json:"ends_at"`
	Timezone             string         `json:"timezone"`
	DailyStart           string         `json:"daily_start"`
	DailyEnd             string         `json:"daily_end"`
	Weekdays             []time.Weekday `json:"weekdays"`
	MinimumSubtotal      int64          `json:"minimum_subtotal_cents"`
	PercentOff           int            `json:"percent_off"`
	MaximumDiscount      int64          `json:"maximum_discount_cents"`
	RequiredTier         string         `json:"required_tier,omitempty"`
	FirstPurchaseOnly    bool           `json:"first_purchase_only"`
	Channels             []string       `json:"channels"`
	CouponCode           string         `json:"coupon_code,omitempty"`
	ApplicableCategories []string       `json:"applicable_categories,omitempty"`
	Stackable            bool           `json:"stackable"`
	Priority             int            `json:"priority"`
}

type CartItem struct {
	SKU            string `json:"sku"`
	Category       string `json:"category"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
}

type Customer struct {
	Tier          string `json:"tier"`
	FirstPurchase bool   `json:"first_purchase"`
}

type QuoteRequest struct {
	At          time.Time
	Channel     string
	Customer    Customer
	Items       []CartItem
	CouponCodes []string
}

type AppliedPromotion struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	DiscountCents int64  `json:"discount_cents"`
}

type PromotionEvaluation struct {
	ID               string   `json:"id"`
	Eligible         bool     `json:"eligible"`
	RejectionReasons []string `json:"rejection_reasons,omitempty"`
}

type Quote struct {
	SubtotalCents int64                 `json:"subtotal_cents"`
	DiscountCents int64                 `json:"discount_cents"`
	TotalCents    int64                 `json:"total_cents"`
	Applied       []AppliedPromotion    `json:"applied_promotions"`
	Evaluations   []PromotionEvaluation `json:"promotion_evaluations"`
}
