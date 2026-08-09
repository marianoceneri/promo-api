package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/marianoceneri/promo-api/internal/httpapi"
	"github.com/marianoceneri/promo-api/internal/repository"
	"github.com/marianoceneri/promo-api/internal/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	repo := repository.NewMemoryPromotionRepository()
	promotionService := service.NewPromotionService(repo)
	quoteService := service.NewQuoteService(repo)
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           httpapi.NewHandler(repo, promotionService, quoteService),
		ReadHeaderTimeout: 5 * time.Second,
	}

	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		slog.Info("promotions API listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-stop.Done()
	ctx, shutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdown()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
