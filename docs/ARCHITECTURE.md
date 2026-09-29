# go-react-saas Architecture

This document explains how `go-react-saas` is organized so new developers and AI agents can quickly navigate the repository, understand responsibilities, and extend the system without breaking its boundaries.

> **Current product: Alba (dating app).** Active feature modules are `auth`, `account`, `profiles`, `photos`, `discovery`, `matching`, `chat`, `notifications`, `safety`; shared seams live in `internal/platform` (`deps`, `storage`, `urlsign`, `ratelimit`, `realtime`, `notify`, `mailer`). See README.md and docs/API.md. Sections below describe the generic boilerplate layout; legacy modules (billing, posts, comments, files, users) are unwired.

## 1. Monorepo Structure

The repository is organized into a small number of top-level directories:

- `apps/`: frontend applications
- `services/`: backend services
- `infra/`: local infrastructure and container orchestration
- `scripts/`: smoke tests, QA helpers, and local automation
- `docs/`: architecture and contributor-facing documentation

At the moment, the main applications are:
- `apps/mobile`: React Native mobile client
- `services/api`: Go API

This layout keeps frontend, backend, infrastructure, and documentation concerns separated while still living in one repository.

## 2. Backend Architecture

The backend uses a feature-first architecture in Go with Gin.

```text
services/api/
├─ cmd/api/main.go
└─ internal/
   ├─ platform/
   │  ├─ config/
   │  ├─ db/
   │  ├─ middleware/
   │  ├─ logger/
   │  └─ errors/
   └─ features/
      ├─ auth/
      ├─ billing/
      ├─ users/
      ├─ files/
      ├─ chat/
      ├─ posts/
      ├─ comments/
      └─ notifications/
```

`cmd/api/main.go` is the composition root. It wires the application together by loading configuration, opening the database, registering feature routes, and starting the HTTP server.

Each feature folder groups everything needed for that domain. Instead of scattering handlers, services, and repositories across separate global folders, the code stays close to the feature it belongs to.

Current notable backend modules:
- `auth`: registration, login, refresh, logout, `/me`, JWT access tokens, refresh-token sessions
- `billing`: checkout and webhook handling for subscription lifecycle
- `users`: tenant-scoped directory access
- `chat`, `posts`, `comments`: tenant-scoped shared content flows

Each backend feature follows the same structure:
- `handler.go`: receives HTTP requests, validates input, and returns HTTP responses
- `service.go`: contains business rules and orchestration
- `repository.go`: contains explicit PostgreSQL queries and persistence logic
- `model.go`: defines domain, request, and response models
- `routes.go`: registers the feature's endpoints with Gin

This pattern improves maintainability because:
- feature code is easier to find
- boundaries are explicit
- business logic stays out of handlers
- SQL stays out of handlers and services
- new modules can follow a predictable template

## 3. Platform Layer

The `platform` layer contains infrastructure shared by all backend features.

- `config/`: loads runtime configuration from environment variables such as `DATABASE_URL`, `JWT_SECRET`, `PORT`, and optional Stripe billing keys
- `db/`: initializes the PostgreSQL connection pool and runs embedded migrations
- `middleware/`: contains reusable HTTP middleware such as auth checks, request metadata handling, and logging hooks
- `logger/`: provides shared logging utilities so features do not each invent their own logging style
- `errors/`: provides shared API error types and helpers for consistent error responses

The purpose of this layer is to centralize technical infrastructure so feature modules stay focused on domain behavior.

## 4. Database Design

The API uses PostgreSQL with explicit SQL queries. There is no ORM.

This is an intentional choice:
- SQL stays visible and reviewable
- query behavior is explicit
- performance characteristics are easier to reason about
- schema changes stay close to the database design

The generic modules rely on core tables such as:
- `users`: account records and identity data
- `organizations`: tenant identity for multi-tenant SaaS isolation
- `subscriptions`: billing state linked to organizations
- `sessions`: refresh-token session persistence
- `posts`: generic user-generated content
- `comments`: responses attached to posts
- `conversations`: chat threads
- `messages`: individual chat messages
- `notifications`: system or user-facing notifications
- `files`: uploaded file metadata and ownership

Related support tables such as `conversation_participants` and `votes` extend these core capabilities.

Multi-tenant isolation is centered on `organizations`:
- users belong to an `organization_id`
- subscriptions belong to an `organization_id`
- tenant-scoped repository queries limit shared content and chat access by organization

## 5. Frontend Architecture

The mobile frontend lives under `apps/mobile/src/`:

```text
apps/mobile/src/
├─ api/
├─ hooks/
├─ components/
├─ screens/
├─ store/
├─ theme/
└─ utils/
```

Responsibilities:
- `api/`: HTTP client functions and endpoint wrappers
- `hooks/`: React Query-powered auth/session hooks
- `components/`: reusable UI building blocks
- `screens/`: route-level UI and screen orchestration
- `store/`: access-token and refresh-token persistence
- `theme/`: design tokens and shared styling primitives
- `utils/`: shared helpers used across the app

Screens should orchestrate the UI, while shared logic belongs in reusable modules such as API clients, helpers, components, and storage utilities. This keeps screens smaller and makes behavior easier to reuse across future clients.

Session behavior is centralized:
- auth state lives in hooks rather than screens
- React Query owns current-user fetching and session restoration
- screens consume hooks and API modules instead of issuing direct network requests

## 6. Adding a New Feature

New backend features should follow the same feature-first pattern:

```text
internal/features/example/
├─ handler.go
├─ service.go
├─ repository.go
├─ model.go
└─ routes.go
```

Recommended process:

1. Add or update the database migration if the feature needs persistence.
2. Create a new folder under `internal/features/<feature>`.
3. Define models in `model.go`.
4. Implement database access in `repository.go`.
5. Implement business logic in `service.go`.
6. Expose HTTP handlers in `handler.go`.
7. Register routes in `routes.go`.
8. Wire the feature from `cmd/api/main.go`.

When adding a feature, preserve the same architectural rules:
- thin handlers
- explicit services
- SQL in repositories
- reusable platform infrastructure
- clear, predictable module boundaries

This consistency is what makes `go-react-saas` scalable as more features are added.
