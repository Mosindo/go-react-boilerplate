FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -o /bin/seed ./cmd/seed

FROM alpine:3.21
RUN addgroup -S app && adduser -S app -G app \
 && mkdir -p /data/uploads && chown -R app:app /data
WORKDIR /app

COPY --from=builder /bin/api ./api
COPY --from=builder /bin/seed ./seed

USER app
ENV UPLOAD_DIR=/data/uploads
EXPOSE 8080

CMD ["./api"]
