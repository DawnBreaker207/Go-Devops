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

RUN apk add --no-cache ca-certificates tzdata wget && \
    adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /out/backend-cp /app/backend-cp
COPY config.yaml /app/config.yaml

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health >/dev/null || exit 1

ENTRYPOINT ["/app/backend-cp"]
