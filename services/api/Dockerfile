FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.21
# alpine already ships ca-certificates-bundle (needed for SMTP over TLS).
RUN addgroup -S app && adduser -S -G app app \
 && mkdir -p /app/data/uploads && chown -R app:app /app/data
WORKDIR /app

COPY --from=builder /out/api /out/seed ./

ENV PORT=8080 \
    UPLOAD_DIR=/app/data/uploads
USER app
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD wget -qO- "http://127.0.0.1:${PORT}/health" >/dev/null || exit 1

CMD ["./api"]
