# ── Build stage ────────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /queue-service .

# ── Runtime stage ──────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk --no-cache add ca-certificates && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

COPY --from=builder /queue-service /queue-service
COPY --from=builder /app/web /web

USER appuser
EXPOSE 8080
CMD ["/queue-service"]
