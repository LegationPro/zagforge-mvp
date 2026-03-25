package zitadel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client defines the interface for Zitadel Management API operations.
// Implementations can be swapped for testing.
type Client interface {
	UpdateUser(ctx context.Context, zitadelUserID string, req UpdateUserRequest) error
	DeleteUser(ctx context.Context, zitadelUserID string) error
	TerminateSession(ctx context.Context, sessionID string) error
}

// UpdateUserRequest contains fields that can be updated on a Zitadel user.
type UpdateUserRequest struct {
	Username string `json:"userName,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

type httpClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient creates a Zitadel Management API client.
// baseURL is the Zitadel issuer URL (e.g. "https://auth.zagforge.com").
// token is a service user Personal Access Token (PAT).
func NewClient(baseURL, token string) Client {
	return &httpClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{},
	}
}

func (c *httpClient) UpdateUser(ctx context.Context, zitadelUserID string, req UpdateUserRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal update request: %w", err)
	}

	url := fmt.Sprintf("%s/v2/users/%s", c.baseURL, zitadelUserID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.do(httpReq)
}

func (c *httpClient) DeleteUser(ctx context.Context, zitadelUserID string) error {
	url := fmt.Sprintf("%s/v2/users/%s", c.baseURL, zitadelUserID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.do(req)
}

func (c *httpClient) TerminateSession(ctx context.Context, sessionID string) error {
	url := fmt.Sprintf("%s/v2/sessions/%s", c.baseURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	return c.do(req)
}

func (c *httpClient) do(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("zitadel api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return fmt.Errorf("zitadel api %s %s: status %d: %s", req.Method, req.URL.Path, resp.StatusCode, body)
}
