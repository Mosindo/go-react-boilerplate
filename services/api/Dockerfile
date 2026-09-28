FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -o /bin/seed ./cmd/seed

FROM alpine:3.21
WORKDIR /app
RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /bin/api ./api
# Demo data tool (refuses to run when APP_ENV=production): docker compose exec api ./seed
COPY --from=builder /bin/seed ./seed

USER app
EXPOSE 8080

CMD ["./api"]
