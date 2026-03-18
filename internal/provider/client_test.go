package provider_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/LegationPro/zagforge-mvp-impl/internal/provider"
)

func validClient(t *testing.T) *provider.ClientHandler {
	t.Helper()
	client, err := provider.NewAPIClient(1, []byte("private-key"), "webhook-secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}
	return provider.NewClientHandler(client)
}

func makeSignature(t *testing.T, secret string, payload []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestNewAPIClient_rejectsEmptyWebhookSecret(t *testing.T) {
	_, err := provider.NewAPIClient(1, []byte("key"), "")
	if err == nil {
		t.Fatal("expected error for empty webhookSecret, got nil")
	}
}

func TestNewAPIClient_rejectsEmptyPrivateKey(t *testing.T) {
	_, err := provider.NewAPIClient(1, nil, "secret")
	if err == nil {
		t.Fatal("expected error for nil privateKey, got nil")
	}
}

func TestNewAPIClient_rejectsEmptyPrivateKeySlice(t *testing.T) {
	_, err := provider.NewAPIClient(1, []byte{}, "secret")
	if err == nil {
		t.Fatal("expected error for empty privateKey slice, got nil")
	}
}

func TestNewAPIClient_succeedsWithValidInputs(t *testing.T) {
	_, err := provider.NewAPIClient(1, []byte("key"), "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateWebhook_validSignature(t *testing.T) {
	ch := validClient(t)
	payload := []byte(`{"action":"push"}`)
	sig := makeSignature(t, "webhook-secret", payload)

	_, err := ch.ValidateWebhook(context.Background(), payload, sig)
	if err != nil {
		t.Fatalf("expected no error for valid signature, got: %v", err)
	}
}

func TestValidateWebhook_invalidSignature(t *testing.T) {
	ch := validClient(t)
	payload := []byte(`{"action":"push"}`)

	_, err := ch.ValidateWebhook(context.Background(), payload, "sha256=badhex")
	if !errors.Is(err, provider.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got: %v", err)
	}
}

func TestValidateWebhook_wrongSecret(t *testing.T) {
	ch := validClient(t)
	payload := []byte(`{"action":"push"}`)
	sig := makeSignature(t, "wrong-secret", payload)

	_, err := ch.ValidateWebhook(context.Background(), payload, sig)
	if !errors.Is(err, provider.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature for wrong secret, got: %v", err)
	}
}

func TestValidateWebhook_emptySignature(t *testing.T) {
	ch := validClient(t)

	_, err := ch.ValidateWebhook(context.Background(), []byte("payload"), "")
	if !errors.Is(err, provider.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature for empty signature, got: %v", err)
	}
}
