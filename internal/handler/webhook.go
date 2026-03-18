package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/LegationPro/zagforge-mvp-impl/internal/provider"
)

// maxPayloadBytes is GitHub's documented maximum webhook payload size.
const maxPayloadBytes = 25 * 1024 * 1024 // 25 MiB

// Compile-time guard: WebhookHandler must satisfy http.Handler.
var _ http.Handler = (*WebhookHandler)(nil)

// WebhookHandler handles POST /internal/webhooks/github.
// It validates the HMAC-SHA256 signature before any processing.
type WebhookHandler struct {
	validator provider.WebhookValidator
}

// NewWebhookHandler constructs a WebhookHandler with the given validator.
func NewWebhookHandler(v provider.WebhookValidator) *WebhookHandler {
	return &WebhookHandler{validator: v}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Fast path: reject missing signature before buffering the body.
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		http.Error(w, "missing signature", http.StatusUnauthorized)
		return
	}

	// Read one byte over the limit so we can detect oversized payloads.
	// io.LimitReader silently truncates without error, so we must check the length ourselves.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes+1))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusInternalServerError)
		return
	}
	if int64(len(body)) > maxPayloadBytes {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	// TODO: pass r.Header.Get("X-GitHub-Event") for event routing once WebhookEvent.EventType is populated
	event, err := h.validator.ValidateWebhook(r.Context(), body, signature)
	if errors.Is(err, provider.ErrInvalidSignature) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "validation error", http.StatusInternalServerError)
		return
	}

	// TODO: dispatch event downstream (job dedup, Cloud Tasks)
	_ = event
	w.WriteHeader(http.StatusOK)
}
