FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S app && adduser -S -G app app \
 && mkdir -p /app/data/uploads && chown -R app:app /app
WORKDIR /app

COPY --from=builder /out/api /out/seed ./

USER app
ENV PORT=8080 UPLOAD_DIR=/app/data/uploads
VOLUME ["/app/data/uploads"]
EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

CMD ["./api"]
