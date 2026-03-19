package handler

import (
	"context"
	"errors"
	"io"
	"net/http"

	github "github.com/LegationPro/zagforge-mvp-impl/shared/go/provider/github"
	"go.uber.org/zap"
)

const maxPayloadBytes = 25 * 1024 * 1024

var _ http.Handler = (*WebhookHandler)(nil)

var supportedEvents = map[string]bool{
	"push": true,
}

// pushHandler receives a validated push event and delivery ID.
// JobService satisfies this interface.
type pushHandler interface {
	HandlePush(ctx context.Context, event github.WebhookEvent, deliveryID string) error
}

type WebhookHandler struct {
	validator github.WebhookValidator
	svc       pushHandler
	log       *zap.Logger
}

func NewWebhookHandler(v github.WebhookValidator, svc pushHandler, log *zap.Logger) *WebhookHandler {
	return &WebhookHandler{validator: v, svc: svc, log: log}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "missing signature", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes+1))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	if int64(len(body)) > maxPayloadBytes {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	event, err := h.validator.ValidateWebhook(r.Context(), body, signature, eventType)
	if errors.Is(err, github.ErrInvalidSignature) {
		h.log.Warn("invalid signature", zap.String("event", eventType))
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	if err != nil {
		h.log.Error("validation error", zap.String("event", eventType), zap.Error(err))
		http.Error(w, "validation error", http.StatusInternalServerError)
		return
	}

	if !supportedEvents[event.EventType] {
		h.log.Info("ignoring unsupported event", zap.String("event", event.EventType))
		w.WriteHeader(http.StatusOK)
		return
	}

	deliveryID := r.Header.Get("X-GitHub-Delivery")
	h.log.Info("dispatching webhook",
		zap.String("event", event.EventType),
		zap.String("repo", event.RepoName),
		zap.String("branch", event.Branch),
		zap.String("commit", event.CommitSHA),
	)

	if err := h.svc.HandlePush(r.Context(), event, deliveryID); err != nil {
		h.log.Error("handle push failed", zap.Error(err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
