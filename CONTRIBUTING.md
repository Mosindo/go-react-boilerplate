# Contributing

Thanks for contributing to Alma.

## Ground Rules

- The product is free for its users: no paywall, subscription, paid boost or artificial limit on core features.
- Server-side authorisation for every rule that matters; never trust the client.
- Prefer incremental changes over broad refactors.
- Update documentation when behavior, architecture, or setup changes.

## Commit Message Conventions

Use Conventional Commits.

Examples:
- `feat(auth): add refresh token validation`
- `fix(chat): handle empty conversation state`
- `docs(readme): clarify local docker workflow`
- `chore(ci): cache Go build artifacts`

Recommended types:
- `feat`
- `fix`
- `docs`
- `refactor`
- `test`
- `chore`
- `build`
- `ci`

## Branch Naming Conventions

Use short, descriptive branch names:
- `feat/user-profiles`
- `fix/chat-pagination`
- `docs/architecture-update`
- `chore/ci-cleanup`

For automated agent work, use the `codex/` prefix.

## Pull Request Workflow

1. Branch from the latest default branch.
2. Keep the scope focused on one logical change.
3. Update tests and documentation when needed.
4. Open a pull request with:
   - a short summary
   - the reason for the change
   - testing notes
   - screenshots or recordings for UI changes when useful
5. Address review feedback with follow-up commits rather than force-pushing away context unless requested.

## Coding Conventions

### Go

- Keep handlers thin.
- Keep business logic in services.
- Keep SQL in repositories (no ORM).
- Use explicit types and clear error handling.
- Run `gofmt` on all edited Go files.
- Avoid adding dependencies unless clearly justified.

### React / React Native

- Use TypeScript.
- Keep screens focused on orchestration and UI.
- Move reusable UI into shared components.
- Keep API calls inside dedicated client modules.
- Keep state and persistence concerns isolated from presentational components.
- Follow the existing project structure before introducing new patterns.

## Required Checks Before Pushing

```bash
cd services/api
export TEST_DATABASE_URL="postgres://alma:alma@localhost:5432/alma_test?sslmode=disable"
test -z "$(gofmt -l .)" && go vet ./... && go test -race -count=1 ./... && go build ./cmd/api
```

```bash
cd apps/mobile
npm ci && npm run typecheck && npm run lint && npm run format:check && npm test
```

For changes touching user flows, also run the browser scenario (API on :18080):

```bash
EXPO_PUBLIC_API_URL=http://localhost:18080 npm run build:web && npm run e2e:web
```

## Documentation Expectations

- Keep `README.md` accurate for onboarding.
- Update `docs/ARCHITECTURE.md` when structure or module boundaries change.
- Record user-facing or release-relevant changes in `CHANGELOG.md`.
