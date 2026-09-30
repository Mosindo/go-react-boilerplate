FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/admin ./cmd/admin

FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
 && addgroup -S app && adduser -S -G app app \
 && mkdir -p /app/data/uploads && chown -R app:app /app/data
WORKDIR /app

COPY --from=builder /out/api /out/seed /out/admin ./

ENV UPLOAD_DIR=/app/data/uploads \
    APP_ENV=production
USER app
EXPOSE 8080
VOLUME ["/app/data/uploads"]

CMD ["./api"]
