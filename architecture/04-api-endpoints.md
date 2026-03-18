# Zagforge — API Endpoints [Phase 2]

## Public API (Clerk API key auth)

`{org}` is the organization `slug`. `{repo}` is the repo `full_name` suffix (e.g., for "LegationPro/zigzag", the repo param is "zigzag").

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/{org}/{repo}/latest` | Latest snapshot for default branch |
| `GET` | `/api/v1/{org}/{repo}/branches/{branch}/latest` | Latest snapshot for a specific branch |
| `GET` | `/api/v1/{org}/{repo}/snapshots` | List snapshot history |
| `GET` | `/api/v1/{org}/{repo}/snapshots/{id}` | Specific snapshot by ID |
| `GET` | `/api/v1/{org}/{repo}/jobs` | List jobs for a repo |

## Internal API

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/internal/webhooks/github` | GitHub App HMAC | Webhook receiver |
| `POST` | `/internal/jobs/start` | Signed job token | Worker reports job started |
| `POST` | `/internal/jobs/complete` | Signed job token | Worker reports job finished |
| `POST` | `/internal/watchdog/timeout` | GCP OIDC token | Cloud Scheduler timeout check |

## GitHub App

| Method | Path | Description |
|---|---|---|
| `GET` | `/auth/github/install` | Redirect to GitHub App installation |
| `GET` | `/auth/github/callback` | Handle installation callback |
