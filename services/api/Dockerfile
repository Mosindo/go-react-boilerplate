FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app \
 && apk add --no-cache ca-certificates \
 && mkdir -p /data/uploads && chown -R app:app /data
WORKDIR /app
COPY --from=builder /out/api /out/seed ./

USER app
ENV PORT=8080 UPLOAD_DIR=/data/uploads
VOLUME ["/data/uploads"]
EXPOSE 8080
CMD ["./api"]
