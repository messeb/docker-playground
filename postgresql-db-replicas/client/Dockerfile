FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates && \
    addgroup -g 99 appuser && \
    adduser -D -u 99 -G appuser appuser
WORKDIR /app
COPY --from=builder /app/main ./main
USER appuser
EXPOSE 8080
CMD ["./main"]
