# Auth Migration: Clerk → Zitadel

## Why

- Clerk forced prebuilt UI, broke our app's look and feel
- Vendor lock-in: forced SDK upgrades, PRO gating for features (e.g. org member limits)
- Coupling auth to Next.js prevents future mobile app and BFF architecture
- Need a standalone auth service that scales independently in a container

## Decision

**Zitadel** — self-hosted on Cloud Run.

- Written in Go, lightweight single binary
- OIDC/SAML/JWT natively, any client (web, mobile, API) speaks standard protocols
- Organizations and multi-tenancy built-in
- Zitadel themselves run their cloud offering on Cloud Run (validated deployment model)
- Open-source (Apache 2.0), no per-user pricing

## Architecture

```
┌──────────────────────┐
│  Cloud Run: Zitadel  │  ← Identity provider (login, SSO, JWT, password reset)
│  auth.zagforge.com   │     Min 1 instance (no cold starts on auth)
└──────────┬───────────┘
           │ OIDC / webhooks / Management API
           │
┌──────────▼───────────┐
│  Cloud Run: API      │  ← Go API (existing)
│  api.zagforge.com    │
│                      │
│  /api/v1/*           │  existing repo/snapshot/token endpoints
│  /api/v1/account/*   │  NEW: profile, sessions, delete account
│  /api/v1/orgs/*      │  NEW: org CRUD, members, invites, audit log
│  /internal/webhooks/ │  NEW: Zitadel event webhooks
│    zitadel            │
└──────────────────────┘
           │
┌──────────▼───────────┐
│  Cloud SQL: Postgres │  ← Shared instance, separate databases
│                      │     (app DB + Zitadel DB)
└──────────────────────┘
```

## User Model

- **Personal workspace**: every user has one, no org required. Repos/tokens/keys scoped via `user_id`.
- **Organizations**: optional team workspaces. A user can create/join many. Resources scoped via `org_id`.
- Resource tables use dual ownership: exactly one of `user_id` or `org_id` must be set.

## User Fields

- Username (required, unique)
- Email (required, unique, verified)
- Password (credential users) or SSO link (Google, GitHub)
- Phone (optional)

## Auth Flows

| Flow | Implementation |
|------|---------------|
| Sign up (credentials) | Zitadel creates user → sends verification email → webhook syncs to DB |
| Sign up (SSO) | Zitadel OIDC redirect → provider login → redirect back → prompt username → webhook syncs |
| Sign in | Zitadel OIDC (Authorization Code + PKCE) |
| Forgot password | Zitadel built-in password reset |
| Change password | Zitadel self-service |
| Change username | API → Zitadel Management API → sync to DB |
| Change email | API → Zitadel Management API → triggers re-verification |
| Delete account | API → Zitadel Management API → cascade delete in DB |
| Sessions | Multi-device, tracked in DB via webhook, user can list/revoke from dashboard |

## New Database Tables

- `users` — synced from Zitadel (zitadel_user_id, username, email, phone, avatar)
- `memberships` — user ↔ org with role (owner/admin/member)
- `sessions` — active sessions per user (device, IP, last active)
- `audit_log` — append-only log of org/personal workspace mutations

## Migration Phases

```
Phase 1  Terraform: Zitadel on Cloud Run + config           ← infra, no code changes
Phase 2  DB migration: users, memberships, sessions,        ← additive, non-breaking
         audit_log; add zitadel_org_id; dual ownership
Phase 3  Auth middleware swap (3 files)                      ← the cutover
Phase 4  Zitadel config: project, apps, SSO providers
Phase 5  User flows: registration, SSO, username prompt
Phase 6  Email: SMTP config in Zitadel + custom welcome
Phase 7  Session dashboard: list/revoke active sessions
Phase 8  Account management: profile, password, delete
Phase 9  Organizations: create, invite, roles, audit log
Phase 10 Cleanup: remove Clerk SDK, secrets, old columns
```

## Files Changed (Go API)

| File | Change |
|------|--------|
| `api/internal/middleware/auth/auth.go` | Replace Clerk JWT verify with Zitadel JWKS-based local verification |
| `api/internal/middleware/auth/orgid.go` | Replace with scope resolver (personal vs. org from JWT/path) |
| `api/internal/middleware/auth/orgscope.go` | Replace `GetOrgByClerkID` with `GetOrgByZitadelID`, support personal scope |
| `api/internal/config/app.go` | Replace `ClerkSecretKey` with `ZitadelIssuerURL` + `ZitadelProjectID` |
| `api/cmd/main.go` | Remove `clerk.SetKey()`, add new routes, update middleware chain |
| `go.mod` | Remove `clerk-sdk-go`, add OIDC/JWKS library |

## Architecture Docs Updated

- `01-overview.md` — tech stack
- `phase1/02-data-model.md` — new tables, dual ownership model
- `phase2/04-api-endpoints.md` — all Clerk JWT → Zitadel OIDC JWT
- `phase2/05-authentication.md` — full rewrite of auth section
- `phase5/15-context-proxy.md` — Query Console auth
- `phase5/16-dashboard.md` — auth flow, routes, workspace switcher, org management
- `phase5/17-cli-upload.md` — CLI token references
- `phase6/18-context-visibility.md` — JWT references, user ID references
- `phase6/19-per-org-cli-credentials.md` — dual ownership for CLI keys
