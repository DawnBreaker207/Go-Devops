# ---------- build stage ----------
FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/backend-cp ./cmd/server

# ---------- runtime stage ----------
FROM alpine:3.20 AS runner

# appuser owns /app/data: ticket emails (mock outbox) and local poster uploads
# are written there; mount a volume on it to keep them.
RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 10001 appuser && \
    mkdir -p /app/data/mail /app/data/uploads && \
    chown -R appuser:appuser /app/data

WORKDIR /app

# No config.yaml in the image: code defaults + environment variables only, so
# dev secrets never ship.
COPY --from=builder /out/backend-cp /app/backend-cp

ENV MAIL_OUTBOX_DIR=/app/data/mail \
    STORAGE_LOCAL_DIR=/app/data/uploads

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health >/dev/null || exit 1

ENTRYPOINT ["/app/backend-cp"]
