package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// WebhookValidator is the minimal interface required by consumers that only validate webhooks.
type WebhookValidator interface {
	ValidateWebhook(ctx context.Context, payload []byte, signature string) (WebhookEvent, error)
}

// Worker is the full provider interface. It embeds WebhookValidator and adds the
// operations needed by the worker container (clone, upload).
type Worker interface {
	WebhookValidator
	GenerateCloneToken(ctx context.Context, installationID int64) (string, error)
	CloneRepo(ctx context.Context, repoURL, ref, token, dst string) error
	ListRepos(ctx context.Context, installationID int64) ([]Repo, error)
}

// APIClient holds provider credentials. Construct with NewAPIClient.
type APIClient struct {
	appID         int64
	privateKey    []byte
	webhookSecret string
}

// NewAPIClient returns a configured APIClient. Returns an error if privateKey or
// webhookSecret is empty — both are required for correct operation at startup.
func NewAPIClient(appID int64, privateKey []byte, webhookSecret string) (*APIClient, error) {
	if len(privateKey) == 0 {
		return nil, errors.New("privateKey must not be empty")
	}
	if webhookSecret == "" {
		return nil, errors.New("webhookSecret must not be empty")
	}
	return &APIClient{
		appID:         appID,
		privateKey:    privateKey,
		webhookSecret: webhookSecret,
	}, nil
}

// ClientHandler wraps an APIClient and satisfies the Worker interface.
type ClientHandler struct {
	client *APIClient
}

// Compile-time guard: ClientHandler must satisfy Worker.
var _ Worker = (*ClientHandler)(nil)

func NewClientHandler(client *APIClient) *ClientHandler {
	if client == nil {
		panic("NewClientHandler: client must not be nil")
	}
	return &ClientHandler{client: client}
}

// ValidateWebhook validates the HMAC-SHA256 signature of a GitHub webhook payload.
// The signature must be in the format "sha256=<hex>" as sent by GitHub in
// the X-Hub-Signature-256 header. Uses constant-time comparison to prevent timing attacks.
// ctx is unused here; retained for interface compatibility.
func (h *ClientHandler) ValidateWebhook(ctx context.Context, payload []byte, signature string) (WebhookEvent, error) {
	mac := hmac.New(sha256.New, []byte(h.client.webhookSecret))
	mac.Write(payload)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return WebhookEvent{}, ErrInvalidSignature
	}

	var event WebhookEvent
	// TODO: parse payload JSON into WebhookEvent fields
	// TODO: populate event.EventType from X-GitHub-Event header (pass via ctx or extra param)
	return event, nil
}

func (h *ClientHandler) GenerateCloneToken(ctx context.Context, installationID int64) (string, error) {
	return "", nil // TODO
}

func (h *ClientHandler) CloneRepo(ctx context.Context, repoURL, ref, token, dst string) error {
	return nil // TODO
}

func (h *ClientHandler) ListRepos(ctx context.Context, installationID int64) ([]Repo, error) {
	return nil, nil // TODO
}
