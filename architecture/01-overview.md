# Zagforge — Overview & Architecture

## Problem

Developers manually run Zigzag CLI, copy markdown output, and paste into AI tools. This is friction that kills adoption. The value of structured codebase context disappears if it requires manual effort on every change.

## Solution

A cloud platform that connects to a developer's GitHub repo, automatically generates Zigzag snapshots on every push, and serves them via a persistent API endpoint that any AI tool can consume.

---

## Implementation Phases

| Phase | Focus | Specs |
|---|---|---|
| **Phase 1** | Project Setup & Data Layer | Repo structure, `go.work`, Docker, local dev, data model, migrations |
| **Phase 2** | Core Loop | API endpoints, authentication, job system, provider/worker, storage |
| **Phase 3** | Infrastructure & Security | Networking (LB, Cloud Armor), rate limiting, security hardening, Terraform |
| **Phase 4** | CI/CD & Production | GitHub Actions pipelines, deployment ops (rollbacks, canary, promotions), monitoring |

Future (not phased): Dashboard UI, billing, GitLab/Bitbucket providers, snapshot diffing, token analytics.

---

## Tech Stack

| Layer | Choice | Rationale |
|---|---|---|
| Backend | Go (Chi router) | Strong stdlib, single binary, excellent concurrency |
| Database | Neon Postgres (free) → Cloud SQL (prod) + sqlc | Neon free tier for dev/early stage; swap to Cloud SQL for production. Standard Postgres connection string, no code changes needed |
| Auth | Clerk | Org management, API keys, Next.js + Go middleware |
| Snapshot engine | Cloud Run Jobs | Per-job isolation, scales to zero, no idle cost |
| API hosting | Cloud Run | Auto-scaling, HTTPS termination, per-second billing |
| Job queue | Cloud Tasks | Managed, retries, scales to zero |
| Snapshot storage | GCS | Native GCP, cheap, prefix-based organization |
| Secret management (prod) | Google Secret Manager | Worker tokens, GitHub App keys, webhook secrets |
| Secret management (dev) | Doppler | Team-shared secrets injection, no `.env` files with real values |
| Config loading | `caarlos0/env` | Struct-based env parsing with validation, defaults, and required fields |
| Dashboard | Next.js (separate deploy, future phase) | Rich interactivity, Clerk SDK |
| Git provider | GitHub first | Provider-agnostic interface for future expansion |

---

## Architecture

```
Developer pushes code
      │
      ▼
GitHub Webhook (push event)
      │
      ▼
Go API (Cloud Run)
  ├── Validate HMAC signature (GitHub App-level secret)
  ├── Parse payload → resolve repo via github_repo_id
  ├── Dedup check (atomic, uses advisory lock):
  │     ├── Queued job exists  → update commit_sha, no new Cloud Tasks task
  │     ├── Running job exists → create new queued job + push to Cloud Tasks
  │     └── No active job      → create new queued job + push to Cloud Tasks
  └── Cloud Tasks task includes: job_id, signed job token
            │
            ▼
      Cloud Tasks (retry: max 3 attempts, exponential backoff)
            │
            ▼
      Cloud Run Job (isolated container, 2 vCPU / 4Gi, timeout: 15min)
        ├── POST /internal/jobs/start (with signed job token)
        ├── Read latest commit_sha from job record
        ├── Shallow clone: git clone --depth 1 --branch <branch> <repo>
        ├── Run Zigzag binary
        ├── Upload snapshot to GCS
        └── POST /internal/jobs/complete (with signed job token)
                  │
                  ▼
            Go API
              ├── Validate signed job token (HMAC over job_id + expiry)
              ├── Idempotency check (skip if already succeeded/failed)
              ├── Update job record (status: succeeded)
              ├── Insert snapshot record (UNIQUE constraint prevents dupes)
              └── Snapshot now servable at GET /api/v1/{org_slug}/{repo}/latest
```

---

## Microservices

Zagforge runs as separate Cloud Run services from day one. Each service is its own Go module, bridged via `go.work` at the repo root:

| Service | Cloud Run Type | Go Module | Description |
|---|---|---|---|
| `api` | Service (always-on) | `zagforge-platform/api` | Public REST API + internal endpoints |
| `worker` | Job (on-demand) | `zagforge-platform/worker` | Snapshot engine — clone, run Zigzag, upload |
| `shared/go` | — (library) | `zagforge-platform/shared/go` | Common Go packages (logger, config, server, etc.) |

Future services (Phase 2+): `dashboard` (Next.js), `billing`, `metrics`.

Each service has its own Docker image, `Dockerfile`, `Dockerfile.dev`, service account, and IAM permissions. Services communicate via HTTP/JSON callbacks (REST). gRPC may be adopted for internal communication in a future phase when stricter contracts or streaming are needed.

---

## API Protocol

| Surface | Protocol | Rationale |
|---|---|---|
| Public API (`/api/v1/*`) | REST/JSON | Universal — AI tools, CLIs, dashboards all speak HTTP/JSON |
| Internal callbacks (`/internal/*`) | REST/JSON | Simple, sufficient for two-call worker lifecycle (start/complete) |
| Future internal (Phase 2+) | gRPC candidate | When service count grows or streaming snapshots are needed |

---

## What Is NOT Phased (Future)

- Dashboard UI (Next.js)
- Billing/subscription management
- GitLab/Bitbucket providers (interface is ready)
- Secret scrubbing
- Snapshot diffing
- Token usage analytics
- Snapshot retention / automatic cleanup
- Structured error codes/enums
- gRPC migration for internal communication
- Multi-region deployment
- Cost monitoring / budget alerts
