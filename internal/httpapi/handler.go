package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/promo-api/internal/domain"
	"example.com/promo-api/internal/repository"
	"example.com/promo-api/internal/service"
)

type Handler struct {
	promotions repository.PromotionRepository
	quotes     *service.QuoteService
}

func NewHandler(promotions repository.PromotionRepository, quotes *service.QuoteService) http.Handler {
	handler := &Handler{promotions: promotions, quotes: quotes}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.health)
	mux.HandleFunc("GET /v1/promotions", handler.listPromotions)
	mux.HandleFunc("POST /v1/quotes", handler.createQuote)
	return mux
}

func (h *Handler) health(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listPromotions(writer http.ResponseWriter, request *http.Request) {
	promotions, err := h.promotions.List(request.Context())
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "could not list promotions")
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"promotions": promotions})
}

type quoteRequest struct {
	At          string            `json:"at"`
	Channel     string            `json:"channel"`
	Customer    domain.Customer   `json:"customer"`
	Items       []domain.CartItem `json:"items"`
	CouponCodes []string          `json:"coupon_codes"`
}

func (h *Handler) createQuote(writer http.ResponseWriter, request *http.Request) {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var input quoteRequest
	if err := decoder.Decode(&input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		writeError(writer, http.StatusBadRequest, "body must contain one JSON object")
		return
	}

	at, err := time.Parse(time.RFC3339, input.At)
	if err != nil {
		writeError(writer, http.StatusBadRequest, "at must be an RFC3339 timestamp")
		return
	}
	quote, err := h.quotes.Quote(request.Context(), domain.QuoteRequest{
		At: at, Channel: input.Channel, Customer: input.Customer, Items: input.Items, CouponCodes: input.CouponCodes,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidQuote) {
			writeError(writer, http.StatusBadRequest, err.Error())
			return
		}
		writeError(writer, http.StatusInternalServerError, "could not calculate quote")
		return
	}
	writeJSON(writer, http.StatusOK, quote)
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
