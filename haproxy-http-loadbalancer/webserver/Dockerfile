FROM alpine:latest

RUN apk --no-cache add ca-certificates && \
    addgroup -g 99 appuser && \
    adduser -D -u 99 -G appuser appuser

ADD ./bin/main /

EXPOSE 8080

CMD ["./main"]