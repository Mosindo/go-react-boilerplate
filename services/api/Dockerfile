FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -o /bin/seed ./cmd/seed

FROM alpine:3.21
RUN adduser -D -u 10001 app && mkdir -p /data/uploads && chown app:app /data/uploads
WORKDIR /app

COPY --from=builder /bin/api ./api
COPY --from=builder /bin/seed ./seed

USER app
ENV UPLOAD_DIR=/data/uploads
VOLUME ["/data/uploads"]
EXPOSE 8080

CMD ["./api"]
