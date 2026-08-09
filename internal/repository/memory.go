package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/marianoceneri/promo-api/internal/domain"
)

var ErrPromotionExists = errors.New("promotion already exists")

type PromotionRepository interface {
	List(context.Context) ([]domain.Promotion, error)
	Create(context.Context, domain.Promotion) error
}

type MemoryPromotionRepository struct {
	mu         sync.RWMutex
	promotions []domain.Promotion
}

func NewMemoryPromotionRepository() *MemoryPromotionRepository {
	return &MemoryPromotionRepository{promotions: seedPromotions()}
}

func NewMemoryPromotionRepositoryWith(promotions []domain.Promotion) *MemoryPromotionRepository {
	copyOfPromotions := clonePromotions(promotions)
	return &MemoryPromotionRepository{promotions: copyOfPromotions}
}

func (r *MemoryPromotionRepository) List(context.Context) ([]domain.Promotion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return clonePromotions(r.promotions), nil
}

func (r *MemoryPromotionRepository) Create(_ context.Context, promotion domain.Promotion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.promotions {
		if existing.ID == promotion.ID {
			return ErrPromotionExists
		}
	}
	r.promotions = append(r.promotions, clonePromotion(promotion))
	return nil
}

func clonePromotions(promotions []domain.Promotion) []domain.Promotion {
	result := make([]domain.Promotion, len(promotions))
	for index, promotion := range promotions {
		result[index] = clonePromotion(promotion)
	}
	return result
}

func clonePromotion(promotion domain.Promotion) domain.Promotion {
	promotion.Weekdays = append([]time.Weekday(nil), promotion.Weekdays...)
	promotion.Channels = append([]string(nil), promotion.Channels...)
	promotion.ApplicableCategories = append([]string(nil), promotion.ApplicableCategories...)
	return promotion
}

func seedPromotions() []domain.Promotion {
	start := mustParseTime("2026-08-01T00:00:00-03:00")
	end := mustParseTime("2026-09-01T00:00:00-03:00")
	allDays := []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday}

	return []domain.Promotion{
		{
			ID: "WEEKDAY20", Name: "Weekday 20%", StartsAt: start, EndsAt: end,
			Timezone: "America/Argentina/Buenos_Aires", DailyStart: "09:00", DailyEnd: "18:00",
			Weekdays:        []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			MinimumSubtotal: 50000, PercentOff: 20, MaximumDiscount: 15000,
			Channels: []string{"web", "app"}, Stackable: false, Priority: 100,
		},
		{
			ID: "APP_NIGHT10", Name: "First purchase at night", StartsAt: start, EndsAt: end,
			Timezone: "America/Argentina/Buenos_Aires", DailyStart: "22:00", DailyEnd: "02:00",
			Weekdays: allDays, MinimumSubtotal: 20000, PercentOff: 10, MaximumDiscount: 5000,
			FirstPurchaseOnly: true, Channels: []string{"app"}, Stackable: true, Priority: 50,
		},
		{
			ID: "VIP_WEEKEND15", Name: "VIP weekend", StartsAt: start, EndsAt: end,
			Timezone: "America/Argentina/Buenos_Aires", DailyStart: "00:00", DailyEnd: "00:00",
			Weekdays: []time.Weekday{time.Saturday, time.Sunday}, MinimumSubtotal: 30000,
			PercentOff: 15, MaximumDiscount: 10000, RequiredTier: "vip",
			Channels: []string{"web", "app"}, Stackable: true, Priority: 70,
		},
		{
			ID: "BOOKS25", Name: "Back to school books", StartsAt: mustParseTime("2026-08-15T00:00:00-03:00"), EndsAt: end,
			Timezone: "America/Argentina/Buenos_Aires", DailyStart: "00:00", DailyEnd: "00:00",
			Weekdays: allDays, MinimumSubtotal: 10000, PercentOff: 25, MaximumDiscount: 8000,
			Channels: []string{"web", "app"}, CouponCode: "BACK2SCHOOL",
			ApplicableCategories: []string{"books"}, Stackable: true, Priority: 60,
		},
	}
}

func mustParseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		panic(err)
	}
	return parsed
}
