# Keycloak provider JARs

The Keycloak image is built from `../Dockerfile`, which downloads the email-OTP
authenticator JAR directly into `/opt/keycloak/providers/` and runs `kc.sh build`.

## Pinned version

- Project : [mesutpiskin/keycloak-2fa-email-authenticator](https://github.com/mesutpiskin/keycloak-2fa-email-authenticator)
- License : Apache-2.0
- SPI tag : `v26.4.2`
- KC tag  : `KC26.6.2` (matches the Keycloak base image)
- File    : `keycloak-2fa-email-authenticator-v26.4.2-KC26.6.2.jar`
- Provider id (used in realm-export.json) : `email-authenticator`

## Bumping the version

When upgrading Keycloak in the Dockerfile, list the JARs published for the new
KC version:

```bash
gh release view <SPI_TAG> --repo mesutpiskin/keycloak-2fa-email-authenticator \
  --json assets --jq '.assets[].name' | grep KC<KC_TAG>
```

Then update `SPI_VERSION` and `KC_VERSION` build args in `../Dockerfile`.

## Verifying the JAR

The release is signed. Each `.jar` has a sibling `.jar.asc` and the project
publishes its GPG key on the release page. To pin a local checksum:

```bash
curl -fsSL "https://github.com/mesutpiskin/keycloak-2fa-email-authenticator/releases/download/v26.4.2/keycloak-2fa-email-authenticator-v26.4.2-KC26.6.2.jar" \
  | sha256sum
```

Add the expected hash to the Dockerfile as a `RUN sha256sum -c` step if you
want supply-chain hardening for this demo.
