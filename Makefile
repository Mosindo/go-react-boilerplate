.PHONY: up down seed test api-test api-build mobile-check

up:
	docker compose --env-file .env -f infra/docker-compose.yml up --build -d

down:
	docker compose --env-file .env -f infra/docker-compose.yml down

seed:
	docker compose --env-file .env -f infra/docker-compose.yml run --rm api ./seed

api-test:
	cd services/api && gofmt -l . && go vet ./... && go test ./...

api-build:
	cd services/api && go build ./cmd/api ./cmd/seed

mobile-check:
	cd apps/mobile && npm run typecheck && npm run lint && npm run format:check && npm test

test: api-test mobile-check
