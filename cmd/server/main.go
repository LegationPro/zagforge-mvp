package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/LegationPro/zagforge-mvp-impl/internal/config"
	"github.com/LegationPro/zagforge-mvp-impl/internal/handler"
	"github.com/LegationPro/zagforge-mvp-impl/internal/provider"
	"github.com/go-chi/chi/v5"
)

func main() {
	c, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	client, err := provider.NewAPIClient(c.App.GithubAppID, []byte(c.App.GithubAppPrivateKey), c.App.GithubAppWebhookSecret)
	if err != nil {
		log.Fatalf("failed to create API client: %v", err)
	}

	ch := provider.NewClientHandler(client)
	wh := handler.NewWebhookHandler(ch)

	r := chi.NewRouter()
	r.Post("/internal/webhooks/github", wh.ServeHTTP)

	srv := &http.Server{
		Addr:    ":" + c.Server.Port,
		Handler: r,
	}

	go func() {
		log.Printf("server listening on :%s", c.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-ctx.Done()

	log.Println("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}

	log.Println("server stopped")
}
