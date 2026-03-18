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

func (m *mockValidator) ValidateWebhook(_ context.Context, _ []byte, _ string, _ string) (provider.WebhookEvent, error) {
	return m.event, m.err
}

// mockDispatcher is a test double for handler.Dispatcher.
type mockDispatcher struct {
	dispatched []provider.WebhookEvent
}

func (m *mockDispatcher) Dispatch(_ context.Context, event provider.WebhookEvent) {
	m.dispatched = append(m.dispatched, event)
}

func post(t *testing.T, h http.Handler, body []byte, signature, eventType string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/internal/webhooks/github", bytes.NewReader(body))
	if signature != "" {
		r.Header.Set("X-Hub-Signature-256", signature)
	}
	if eventType != "" {
		r.Header.Set("X-GitHub-Event", eventType)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func newHandler(v *mockValidator, d *mockDispatcher) *handler.WebhookHandler {
	return handler.NewWebhookHandler(v, d)
}

func TestServeHTTP_missingSignature_returns401(t *testing.T) {
	h := newHandler(&mockValidator{}, &mockDispatcher{})
	w := post(t, h, []byte(`{}`), "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestServeHTTP_invalidSignature_returns401(t *testing.T) {
	h := newHandler(&mockValidator{err: provider.ErrInvalidSignature}, &mockDispatcher{})
	w := post(t, h, []byte(`{}`), "sha256=bad", "push")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestServeHTTP_validSignature_returns200(t *testing.T) {
	h := newHandler(&mockValidator{event: provider.WebhookEvent{}}, &mockDispatcher{})
	w := post(t, h, []byte(`{"action":"push"}`), "sha256=valid", "push")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestServeHTTP_validationError_returns500(t *testing.T) {
	h := newHandler(&mockValidator{err: errors.New("unexpected internal error")}, &mockDispatcher{})
	w := post(t, h, []byte(`{}`), "sha256=anything", "push")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestServeHTTP_unsupportedEvent_returns200(t *testing.T) {
	d := &mockDispatcher{}
	h := newHandler(&mockValidator{event: provider.WebhookEvent{EventType: "ping"}}, d)
	w := post(t, h, []byte(`{}`), "sha256=valid", "ping")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for unsupported event, got %d", w.Code)
	}
	if len(d.dispatched) != 0 {
		t.Errorf("expected no dispatch for unsupported event, got %d", len(d.dispatched))
	}
}

func TestServeHTTP_pushEvent_dispatches(t *testing.T) {
	event := provider.WebhookEvent{EventType: "push", RepoName: "org/repo", Branch: "main"}
	d := &mockDispatcher{}
	h := newHandler(&mockValidator{event: event}, d)
	w := post(t, h, []byte(`{}`), "sha256=valid", "push")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for push event, got %d", w.Code)
	}
	if len(d.dispatched) != 1 {
		t.Fatalf("expected 1 dispatch, got %d", len(d.dispatched))
	}
	if d.dispatched[0].RepoName != "org/repo" {
		t.Errorf("expected dispatched RepoName %q, got %q", "org/repo", d.dispatched[0].RepoName)
	}
}

func TestServeHTTP_oversizedBody_returns413(t *testing.T) {
	h := newHandler(&mockValidator{}, &mockDispatcher{})
	// Allocates 25 MiB + 1 byte in memory. This is intentional: the handler must buffer
	// up to maxPayloadBytes+1 before it can detect an oversized payload (io.LimitReader
	// truncates silently, so we read one byte over the limit then check the length).
	bigBody := bytes.Repeat([]byte("x"), 25*1024*1024+1)
	w := post(t, h, bigBody, "sha256=anything", "push")
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}
