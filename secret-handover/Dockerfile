# syntax=docker/dockerfile:1.4
FROM alpine:3.20

RUN apk add --no-cache curl

# Create config directory
RUN mkdir -p /etc/app

# Fetch the private asset using a secret — the token is mounted read-only
# for this single RUN step only and is never written to any image layer.
RUN --mount=type=secret,id=registry_token \
    curl -sSf \
      -H "Authorization: Bearer $(cat /run/secrets/registry_token)" \
      http://host.docker.internal:19080/private-config.txt \
      -o /etc/app/private-config.txt

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

CMD ["/usr/local/bin/entrypoint.sh"]
