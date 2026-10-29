# Developer shortcuts. Requires: Go 1.24+, Node 22+, Docker.
COMPOSE = docker compose --env-file .env -f infra/docker-compose.yml
TEST_DB ?= postgres://app:app@localhost:5432/app_test?sslmode=disable

.PHONY: help up down logs seed db test test-api test-mobile lint fmt build smoke

help:
	@echo "make up         start postgres + api in Docker"
	@echo "make seed       load demo accounts (development only)"
	@echo "make test       run API and mobile tests"
	@echo "make lint       gofmt/vet + eslint + prettier check"
	@echo "make build      build the API binary and the mobile bundle"
	@echo "make smoke      end-to-end journey against a running API (API_URL=...)"

up:
	$(COMPOSE) up --build -d

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f api

seed:
	$(COMPOSE) run --rm api ./seed

db:
	$(COMPOSE) up -d postgres

# Integration tests wipe DATABASE_URL_TEST: use a dedicated database.
test-api:
	cd services/api && DATABASE_URL_TEST="$(TEST_DB)" go test -count=1 ./...

test-mobile:
	cd apps/mobile && npm test -- --ci

test: test-api test-mobile

lint:
	cd services/api && test -z "$$(gofmt -l .)" && go vet ./...
	cd apps/mobile && npm run typecheck && npm run lint && npm run format:check

fmt:
	cd services/api && gofmt -w .
	cd apps/mobile && npm run format

build:
	cd services/api && go build -o bin/api ./cmd/api
	cd apps/mobile && npm run build

smoke:
	node scripts/smoke.mjs
