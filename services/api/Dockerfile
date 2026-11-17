FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.21
# The base image already ships the CA bundle (SMTP TLS) and busybox wget (healthcheck).
RUN addgroup -S lueur && adduser -S -G lueur lueur \
 && mkdir -p /data/uploads && chown -R lueur:lueur /data
WORKDIR /app

COPY --from=builder /out/api /out/seed ./

USER lueur
ENV PORT=8080 UPLOAD_DIR=/data/uploads
VOLUME ["/data/uploads"]
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --retries=5 CMD wget -qO- "http://127.0.0.1:${PORT}/health" >/dev/null || exit 1

CMD ["./api"]
