package handler_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LegationPro/zagforge-mvp-impl/internal/handler"
	"github.com/LegationPro/zagforge-mvp-impl/internal/provider"
)

// mockValidator is a test double for provider.WebhookValidator.
type mockValidator struct {
	event provider.WebhookEvent
	err   error
}

func (m *mockValidator) ValidateWebhook(_ context.Context, _ []byte, _ string) (provider.WebhookEvent, error) {
	return m.event, m.err
}

func post(t *testing.T, h http.Handler, body []byte, signature string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/internal/webhooks/github", bytes.NewReader(body))
	if signature != "" {
		r.Header.Set("X-Hub-Signature-256", signature)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestServeHTTP_missingSignature_returns401(t *testing.T) {
	h := handler.NewWebhookHandler(&mockValidator{})
	w := post(t, h, []byte(`{}`), "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestServeHTTP_invalidSignature_returns401(t *testing.T) {
	h := handler.NewWebhookHandler(&mockValidator{err: provider.ErrInvalidSignature})
	w := post(t, h, []byte(`{}`), "sha256=bad")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestServeHTTP_validSignature_returns200(t *testing.T) {
	h := handler.NewWebhookHandler(&mockValidator{event: provider.WebhookEvent{}})
	w := post(t, h, []byte(`{"action":"push"}`), "sha256=valid")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServeHTTP_validationError_returns500(t *testing.T) {
	h := handler.NewWebhookHandler(&mockValidator{err: errors.New("unexpected internal error")})
	w := post(t, h, []byte(`{}`), "sha256=anything")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestServeHTTP_oversizedBody_returns413(t *testing.T) {
	h := handler.NewWebhookHandler(&mockValidator{})
	// Allocates 25 MiB + 1 byte in memory. This is intentional: the handler must buffer
	// up to maxPayloadBytes+1 before it can detect an oversized payload (io.LimitReader
	// truncates silently, so we read one byte over the limit then check the length).
	bigBody := bytes.Repeat([]byte("x"), 25*1024*1024+1)
	w := post(t, h, bigBody, "sha256=anything")
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}
