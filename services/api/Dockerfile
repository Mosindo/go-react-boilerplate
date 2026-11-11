FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/seed ./cmd/seed

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S app && adduser -S -G app app \
 && mkdir -p /data/uploads && chown -R app:app /data
WORKDIR /app

COPY --from=builder /bin/api /bin/seed ./

ENV UPLOAD_DIR=/data/uploads \
    PORT=8080
VOLUME ["/data/uploads"]
USER app
EXPOSE 8080

CMD ["./api"]
