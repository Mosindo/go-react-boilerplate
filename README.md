# go-react-saas

`go-react-saas` is a fullstack monorepo whose product is now a 100% free dating app: a Go API, PostgreSQL database, Docker-based infrastructure and a React Native (Expo) client. There are no payments, no paywall and no quotas on likes. The HTTP contract is documented in [docs/API.md](docs/API.md).

## Project Overview

This repository provides a production-minded starting point for fullstack products that need:
- a typed mobile frontend
- a modular Go API
- explicit PostgreSQL persistence
- session-based authentication with refresh-token rotation
- dating features: profiles, photos, discovery, matches, chat (REST + WebSocket), notifications, blocks and reports
- containerized local development
- predictable smoke and QA scripts

The codebase is intentionally generic. Core modules can be composed and extended without assuming a single business vertical.

The default Docker Compose project name is `go-react-saas`, which keeps generated container, network, and volume names stable across environments.

## Stack Description

### Backend
- Go
- Gin
- PostgreSQL
- `pgxpool`
- JWT authentication with access tokens + refresh tokens
- explicit SQL queries without an ORM

### Frontend
- React Native with Expo
- React-compatible structure for additional clients
- TypeScript
- React Navigation
- React Query for auth/session orchestration

### Infrastructure
- Docker
- Docker Compose
- GitHub Actions for QA automation

## Monorepo Structure

```text
.
├─ apps/
│  └─ mobile/
│     ├─ App.tsx
│     ├─ babel.config.js
│     ├─ e2e/
│     ├─ scripts/
│     └─ src/
│        ├─ api/
│        ├─ components/
│        ├─ screens/
│        ├─ store/
│        ├─ theme/
│        └─ utils/
├─ docs/
│  └─ ARCHITECTURE.md
├─ infra/
│  └─ docker-compose.yml
├─ scripts/
│  ├─ qa-lite.ps1
│  ├─ smoke-all.ps1
│  └─ smoke-api.ps1
├─ services/
│  └─ api/
│     ├─ cmd/api/
│     └─ internal/
│        ├─ features/
│        └─ platform/
├─ AGENTS.md
├─ CHANGELOG.md
├─ CODE_OF_CONDUCT.md
├─ CONTRIBUTING.md
└─ README.md
```

## Backend Architecture Overview

The API uses a feature-first structure with a small shared platform layer.

- `services/api/cmd/api/main.go` is the composition root
- `services/api/internal/platform` contains cross-cutting concerns such as config, database setup, middleware, logging, and shared errors
- `services/api/internal/features` contains domain modules such as `auth`, `profiles`, `photos`, `discovery`, `matches`, `chat`, `realtime`, `safety` and `notifications` (legacy `users`, `posts`, `comments`, `files`, `billing` are unwired, see docs/PENDING_REMOVAL.md)

Each feature follows the same shape:
- `handler.go`
- `service.go`
- `repository.go`
- `model.go`
- `routes.go`

This keeps routing, business logic, and SQL separated while making new modules easy to add.

For a deeper breakdown, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Frontend Architecture Overview

The mobile app is organized around generic product surfaces:
- authentication
- home feed
- chat
- notifications
- profile
- session restoration and refresh-token aware auth state

Within `apps/mobile/src`:
- `api` contains REST clients
- `hooks` contains React Query-powered auth/session hooks
- `screens` contains route-level UI
- `shared/ui` contains reusable UI primitives and tokens
- `shared/layout` contains global layout and navigation shells
- `shared/feedback` contains reusable loading, error, and empty states
- `store` contains token/session persistence
- `theme` contains shared design tokens
- `utils` contains shared helpers

The current frontend shell is designed to be reused and extended rather than tied to a single product concept.

## How To Run Locally

## Prerequisites

- Node.js 20+
- npm
- Go 1.23+ with toolchain support
- Docker Desktop or Docker Engine

## 1. Prepare environment

Copy .env.example to .env and provide a strong JWT_SECRET before starting the stack.

## 2. Start PostgreSQL and the API with Docker

PowerShell:

```powershell
$env:JWT_SECRET='replace-with-a-random-secret-at-least-32-characters-long'
docker compose up --build -d
```

Bash:

```bash
export JWT_SECRET='replace-with-a-random-secret-at-least-32-characters-long'
docker compose up --build -d
```

The API is exposed on `http://localhost:18080`.

Optional runtime variables: `APP_ENV` (`development` default, `test`, `production`), `ALLOWED_ORIGINS` (CORS and WebSocket origins), `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` (SMTP is mandatory when `APP_ENV=production`; otherwise password-reset emails are written to the API log). Demo data: see [docs/SEED.md](docs/SEED.md).

Health check:

```bash
curl http://localhost:18080/health
```

## 3. Run the API directly

PowerShell:

```powershell
cd services/api
$env:DATABASE_URL='postgresql://app:app@localhost:5432/app?sslmode=disable'
$env:JWT_SECRET='replace-with-a-random-secret-at-least-32-characters-long'
go run ./cmd/api
```

Bash:

```bash
cd services/api
export DATABASE_URL='postgresql://app:app@localhost:5432/app?sslmode=disable'
export JWT_SECRET='replace-with-a-random-secret-at-least-32-characters-long'
go run ./cmd/api
```

## 4. Start the mobile app

```bash
cd apps/mobile
npm install
npm run start
```

Other launch targets:

```bash
npm run android
npm run ios
npm run web
```

For physical devices, set:

```bash
EXPO_PUBLIC_API_URL=http://<LAN_IP>:18080
```

## 5. Run checks

Backend:

```bash
cd services/api
go test ./...
go build ./cmd/api
```

Mobile:

```bash
cd apps/mobile
npx tsc --noEmit
npx expo export --platform android --output-dir .expo-audit
```

Repository-wide QA:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\qa-lite.ps1 -ApiBaseUrl http://localhost:18080
```

## Payments

There are none. The former Stripe/billing module is unwired from the API and listed in [docs/PENDING_REMOVAL.md](docs/PENDING_REMOVAL.md) for the owner to validate its removal.

## How To Add New Features

Use the existing vertical-slice pattern and keep naming generic.

### Backend
1. Add or update a migration in `services/api/internal/platform/db/migrations`.
2. Create a new feature folder under `services/api/internal/features/<feature>`.
3. Implement `model.go`, `repository.go`, `service.go`, `handler.go`, and `routes.go`.
4. Keep SQL in repositories and HTTP concerns in handlers.
5. Register the feature in `services/api/cmd/api/main.go`.
6. Add tests before expanding the public API surface.

Current backend foundations include:
- sessions for refresh-token lifecycle (revocable, checked on every request)
- a `schema_migrations` table: migrations are append-only and applied once
- `platform/httpx` for the shared error body `{error, code}` and pagination cursors
- `platform/middleware` for auth, in-memory rate limits and body limits

### Frontend
1. Add API client functions in `apps/mobile/src/api`.
2. Add or extend screens in `apps/mobile/src/screens`.
3. Extract reusable UI into `apps/mobile/src/shared/ui`.
4. Keep layout concerns in `shared/layout` and stateful feedback in `shared/feedback`.
5. Keep tokens and storage concerns centralized in `theme` and `store`.
6. Re-run TypeScript and smoke checks after changes.

## Additional Documentation

- [CONTRIBUTING.md](CONTRIBUTING.md)
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- [CHANGELOG.md](CHANGELOG.md)
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

