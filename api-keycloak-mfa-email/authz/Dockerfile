# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /authz ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk --no-cache add ca-certificates && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

COPY --from=builder /authz /authz

USER appuser
EXPOSE 9000
CMD ["/authz"]
