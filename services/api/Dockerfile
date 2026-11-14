FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/seed ./cmd/seed

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app \
 && apk add --no-cache ca-certificates \
 && mkdir -p /app/data/uploads && chown -R app:app /app
WORKDIR /app

COPY --from=builder /bin/api ./api
COPY --from=builder /bin/seed ./seed

USER app
ENV UPLOAD_DIR=/app/data/uploads
VOLUME ["/app/data"]
EXPOSE 8080

CMD ["./api"]
