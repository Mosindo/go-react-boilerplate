FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/seed ./cmd/seed

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget \
 && addgroup -S app && adduser -S -G app app \
 && mkdir -p /data/uploads && chown -R app:app /data
WORKDIR /app

COPY --from=builder /bin/api ./api
COPY --from=builder /bin/seed ./seed

# Photos live outside the image: mount a volume (or switch the Storage implementation to object storage).
ENV UPLOAD_DIR=/data/uploads
VOLUME ["/data/uploads"]
USER app
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
  CMD wget -qO- "http://127.0.0.1:${PORT:-8080}/health" >/dev/null || exit 1

CMD ["./api"]
