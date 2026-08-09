package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/marianoceneri/promo-api/internal/domain"
	"github.com/marianoceneri/promo-api/internal/repository"
)

var ErrInvalidPromotion = errors.New("invalid promotion")

type PromotionService struct {
	promotions repository.PromotionRepository
}

func NewPromotionService(promotions repository.PromotionRepository) *PromotionService {
	return &PromotionService{promotions: promotions}
}

func (s *PromotionService) Create(ctx context.Context, promotion domain.Promotion) (domain.Promotion, error) {
	if err := validatePromotion(promotion); err != nil {
		return domain.Promotion{}, err
	}
	if err := s.promotions.Create(ctx, promotion); err != nil {
		return domain.Promotion{}, fmt.Errorf("create promotion: %w", err)
	}
	return promotion, nil
}

func validatePromotion(promotion domain.Promotion) error {
	if strings.TrimSpace(promotion.ID) == "" || strings.TrimSpace(promotion.Name) == "" {
		return fmt.Errorf("%w: id and name are required", ErrInvalidPromotion)
	}
	if promotion.StartsAt.IsZero() || promotion.EndsAt.IsZero() || !promotion.StartsAt.Before(promotion.EndsAt) {
		return fmt.Errorf("%w: starts_at must precede ends_at", ErrInvalidPromotion)
	}
	if _, err := time.LoadLocation(promotion.Timezone); err != nil {
		return fmt.Errorf("%w: timezone is invalid", ErrInvalidPromotion)
	}
	if _, err := minuteOfDay(promotion.DailyStart); err != nil {
		return fmt.Errorf("%w: daily_start must use HH:MM", ErrInvalidPromotion)
	}
	if _, err := minuteOfDay(promotion.DailyEnd); err != nil {
		return fmt.Errorf("%w: daily_end must use HH:MM", ErrInvalidPromotion)
	}
	if len(promotion.Weekdays) == 0 {
		return fmt.Errorf("%w: at least one weekday is required", ErrInvalidPromotion)
	}
	seenDays := make(map[time.Weekday]struct{}, len(promotion.Weekdays))
	for _, weekday := range promotion.Weekdays {
		if weekday < time.Sunday || weekday > time.Saturday {
			return fmt.Errorf("%w: weekdays must be between 0 and 6", ErrInvalidPromotion)
		}
		if _, exists := seenDays[weekday]; exists {
			return fmt.Errorf("%w: weekdays cannot repeat", ErrInvalidPromotion)
		}
		seenDays[weekday] = struct{}{}
	}
	if promotion.MinimumSubtotal < 0 || promotion.MaximumDiscount < 0 {
		return fmt.Errorf("%w: monetary limits cannot be negative", ErrInvalidPromotion)
	}
	if promotion.PercentOff < 1 || promotion.PercentOff > 100 {
		return fmt.Errorf("%w: percent_off must be between 1 and 100", ErrInvalidPromotion)
	}
	if len(promotion.Channels) == 0 {
		return fmt.Errorf("%w: at least one channel is required", ErrInvalidPromotion)
	}
	for _, channel := range promotion.Channels {
		if strings.TrimSpace(channel) == "" {
			return fmt.Errorf("%w: channels cannot be blank", ErrInvalidPromotion)
		}
	}
	return nil
}
