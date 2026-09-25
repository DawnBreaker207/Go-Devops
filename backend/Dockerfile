# syntax=docker/dockerfile:1
# ---------- deps stage ----------
FROM golang:1.26-alpine AS deps

WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
# `all` (not bare): test-only transitive deps must also live IN the image,
# otherwise `go test` re-downloads them at container runtime.
RUN go mod download all

# ---------- test stage (CI) ----------
FROM deps AS test

COPY . .

# Tests run when the container starts (`docker compose run`), so this stage
# works with any builder — no custom build network needed. Fails CI when
# vet or any test fails (nonzero exit propagates through `run`).
CMD ["sh", "-c", "go vet ./... && go test ./... -count=1"]

# ---------- build stage ----------
FROM deps AS builder

COPY . .
# GOCACHE mount only speeds rebuilds; modules already live in the image.
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/backend-cp ./cmd/server

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
  CMD wget -qO- http://localhost:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/backend-cp"]
