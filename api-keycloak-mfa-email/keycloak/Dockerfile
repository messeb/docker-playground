# ── Builder stage ─────────────────────────────────────────────────────────────
# Pulls the community email-OTP authenticator JAR into /opt/keycloak/providers
# and runs `kc.sh build` so the provider is registered at image build time
# (the supported way to ship Keycloak extensions on 26.x).
#
# SPI source: https://github.com/mesutpiskin/keycloak-2fa-email-authenticator
# License   : Apache-2.0
# Provider  : email-authenticator (used in browser-mfa flow in realm-export.json)
FROM quay.io/keycloak/keycloak:26.6.2 AS builder

ARG SPI_VERSION=v26.4.2
ARG KC_VERSION=KC26.6.2
ADD --chmod=644 \
    https://github.com/mesutpiskin/keycloak-2fa-email-authenticator/releases/download/${SPI_VERSION}/keycloak-2fa-email-authenticator-${SPI_VERSION}-${KC_VERSION}.jar \
    /opt/keycloak/providers/keycloak-2fa-email-authenticator.jar

RUN /opt/keycloak/bin/kc.sh build

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM quay.io/keycloak/keycloak:26.6.2

COPY --from=builder /opt/keycloak/ /opt/keycloak/

# Realm is mounted at runtime from compose.yml so iteration doesn't require rebuild.
