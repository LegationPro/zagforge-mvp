package provider

import "errors"

// ErrInvalidSignature is returned by ValidateWebhook when the HMAC signature does not match.
var ErrInvalidSignature = errors.New("invalid webhook signature")

type ActionType string

// WebhookEvent is the parsed result of a validated webhook payload.
// Validation would not be necessary on these fields since they're returned from GitHub already.
type WebhookEvent struct {
	EventType string // value of X-GitHub-Event header; populated in a future task
	Action    ActionType
	RepoID    int64
	RepoName  string
	Branch    string
	CommitSHA string
}

type Repo struct {
	ID            int64
	FullName      string
	DefaultBranch string
}
