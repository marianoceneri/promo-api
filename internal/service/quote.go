package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"example.com/promo-api/internal/domain"
	"example.com/promo-api/internal/repository"
)

var ErrInvalidQuote = errors.New("invalid quote request")

type QuoteService struct {
	promotions repository.PromotionRepository
}

func NewQuoteService(promotions repository.PromotionRepository) *QuoteService {
	return &QuoteService{promotions: promotions}
}

type candidate struct {
	promotion domain.Promotion
	discount  int64
}

func (s *QuoteService) Quote(ctx context.Context, request domain.QuoteRequest) (domain.Quote, error) {
	subtotal, err := validateAndSubtotal(request)
	if err != nil {
		return domain.Quote{}, err
	}

	promotions, err := s.promotions.List(ctx)
	if err != nil {
		return domain.Quote{}, fmt.Errorf("list promotions: %w", err)
	}

	evaluations := make([]domain.PromotionEvaluation, 0, len(promotions))
	candidates := make([]candidate, 0, len(promotions))
	for _, promotion := range promotions {
		reasons := rejectionReasons(promotion, request, subtotal)
		evaluation := domain.PromotionEvaluation{ID: promotion.ID, Eligible: len(reasons) == 0, RejectionReasons: reasons}
		evaluations = append(evaluations, evaluation)
		if len(reasons) != 0 {
			continue
		}

		base := applicableSubtotal(promotion, request.Items)
		discount := base * int64(promotion.PercentOff) / 100
		if promotion.MaximumDiscount > 0 && discount > promotion.MaximumDiscount {
			discount = promotion.MaximumDiscount
		}
		if discount > 0 {
			candidates = append(candidates, candidate{promotion: promotion, discount: discount})
		}
	}

	applied := selectPromotions(candidates, subtotal)
	discount := int64(0)
	for _, item := range applied {
		discount += item.DiscountCents
	}
	if discount > subtotal {
		discount = subtotal
	}

	return domain.Quote{
		SubtotalCents: subtotal,
		DiscountCents: discount,
		TotalCents:    subtotal - discount,
		Applied:       applied,
		Evaluations:   evaluations,
	}, nil
}

func validateAndSubtotal(request domain.QuoteRequest) (int64, error) {
	if request.At.IsZero() {
		return 0, fmt.Errorf("%w: at is required", ErrInvalidQuote)
	}
	if strings.TrimSpace(request.Channel) == "" {
		return 0, fmt.Errorf("%w: channel is required", ErrInvalidQuote)
	}
	if len(request.Items) == 0 {
		return 0, fmt.Errorf("%w: at least one item is required", ErrInvalidQuote)
	}

	var subtotal int64
	for index, item := range request.Items {
		if item.SKU == "" || item.Category == "" || item.Quantity <= 0 || item.UnitPriceCents <= 0 {
			return 0, fmt.Errorf("%w: item %d is invalid", ErrInvalidQuote, index)
		}
		subtotal += int64(item.Quantity) * item.UnitPriceCents
	}
	return subtotal, nil
}

func rejectionReasons(promotion domain.Promotion, request domain.QuoteRequest, subtotal int64) []string {
	reasons := make([]string, 0, 4)
	if request.At.Before(promotion.StartsAt) || !request.At.Before(promotion.EndsAt) {
		reasons = append(reasons, "outside_date_range")
	}

	localTime, locationErr := promotionLocalTime(promotion, request.At)
	if locationErr != nil {
		reasons = append(reasons, "invalid_promotion_timezone")
	} else {
		if !containsWeekday(promotion.Weekdays, localTime.Weekday()) {
			reasons = append(reasons, "weekday_not_allowed")
		}
		if !insideDailyWindow(localTime, promotion.DailyStart, promotion.DailyEnd) {
			reasons = append(reasons, "outside_daily_window")
		}
	}

	if subtotal < promotion.MinimumSubtotal {
		reasons = append(reasons, "subtotal_below_minimum")
	}
	if promotion.RequiredTier != "" && !strings.EqualFold(promotion.RequiredTier, request.Customer.Tier) {
		reasons = append(reasons, "customer_tier_not_allowed")
	}
	if promotion.FirstPurchaseOnly && !request.Customer.FirstPurchase {
		reasons = append(reasons, "first_purchase_required")
	}
	if !containsFold(promotion.Channels, request.Channel) {
		reasons = append(reasons, "channel_not_allowed")
	}
	if promotion.CouponCode != "" && !containsFold(request.CouponCodes, promotion.CouponCode) {
		reasons = append(reasons, "coupon_required")
	}
	if len(promotion.ApplicableCategories) > 0 && applicableSubtotal(promotion, request.Items) == 0 {
		reasons = append(reasons, "no_eligible_items")
	}
	return reasons
}

func promotionLocalTime(promotion domain.Promotion, at time.Time) (time.Time, error) {
	location, err := time.LoadLocation(promotion.Timezone)
	if err != nil {
		return time.Time{}, err
	}
	return at.In(location), nil
}

func insideDailyWindow(at time.Time, startText, endText string) bool {
	start, startErr := minuteOfDay(startText)
	end, endErr := minuteOfDay(endText)
	if startErr != nil || endErr != nil {
		return false
	}
	if start == end {
		return true
	}
	minute := at.Hour()*60 + at.Minute()
	if start < end {
		return minute >= start && minute < end
	}
	return minute >= start || minute < end
}

func minuteOfDay(value string) (int, error) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, errors.New("invalid daily time")
	}
	hour, hourErr := strconv.Atoi(parts[0])
	minute, minuteErr := strconv.Atoi(parts[1])
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, errors.New("invalid daily time")
	}
	return hour*60 + minute, nil
}

func applicableSubtotal(promotion domain.Promotion, items []domain.CartItem) int64 {
	if len(promotion.ApplicableCategories) == 0 {
		var total int64
		for _, item := range items {
			total += int64(item.Quantity) * item.UnitPriceCents
		}
		return total
	}

	var total int64
	for _, item := range items {
		if containsFold(promotion.ApplicableCategories, item.Category) {
			total += int64(item.Quantity) * item.UnitPriceCents
		}
	}
	return total
}

func selectPromotions(candidates []candidate, subtotal int64) []domain.AppliedPromotion {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].promotion.Priority == candidates[j].promotion.Priority {
			return candidates[i].promotion.ID < candidates[j].promotion.ID
		}
		return candidates[i].promotion.Priority > candidates[j].promotion.Priority
	})

	var exclusive *candidate
	stackable := make([]candidate, 0, len(candidates))
	var stackableDiscount int64
	for index := range candidates {
		item := candidates[index]
		if item.promotion.Stackable {
			stackable = append(stackable, item)
			stackableDiscount += item.discount
			continue
		}
		if exclusive == nil || item.discount > exclusive.discount ||
			(item.discount == exclusive.discount && item.promotion.Priority > exclusive.promotion.Priority) {
			copyOfItem := item
			exclusive = &copyOfItem
		}
	}
	if stackableDiscount > subtotal {
		stackableDiscount = subtotal
	}

	if exclusive != nil && exclusive.discount >= stackableDiscount {
		return []domain.AppliedPromotion{{
			ID: exclusive.promotion.ID, Name: exclusive.promotion.Name, DiscountCents: exclusive.discount,
		}}
	}

	result := make([]domain.AppliedPromotion, 0, len(stackable))
	remaining := subtotal
	for _, item := range stackable {
		discount := item.discount
		if discount > remaining {
			discount = remaining
		}
		result = append(result, domain.AppliedPromotion{
			ID: item.promotion.ID, Name: item.promotion.Name, DiscountCents: discount,
		})
		remaining -= discount
		if remaining == 0 {
			break
		}
	}
	return result
}

func containsWeekday(values []time.Weekday, expected time.Weekday) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func containsFold(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(value, expected) {
			return true
		}
	}
	return false
}
