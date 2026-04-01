# 🔐 Secure Docker Container

A multi-stage Docker build for a Spring Boot application using Java 21, incorporating security best practices such as non-root users, read-only file permissions, and lean runtime images.

---

## 📋 Table of Contents

- [Overview](#-overview)
- [Build Stage](#-build-stage)
- [Runtime Stage](#-runtime-stage)
- [Usage](#-usage)

---

## 🏗️ Overview

This [`Dockerfile`](Dockerfile) uses a two-stage build:

| Stage | Image | Purpose |
|-------|-------|---------|
| Build | `gradle:9.4.1-jdk25` | Compile and package the application |
| Runtime | `eclipse-temurin:25-jre-alpine` | Run the application in a minimal, secure environment |

---

## 🔨 Build Stage

Building inside a Docker container provides a clean, isolated, and reproducible environment — protecting the host and the final artifact.

```dockerfile
FROM gradle:9.4.1-jdk25 AS build

WORKDIR /app

COPY java-service/gradle/ ./gradle
COPY java-service/gradlew .
COPY java-service/build.gradle .
COPY java-service/settings.gradle .

RUN ./gradlew dependencies --no-daemon

COPY java-service .
RUN ./gradlew bootJar --no-daemon
```

### Security Benefits

#### 🧹 Clean Environment

- Each build starts from a fresh, known state — no residue from previous builds or host-side tools.
- Host-installed tools (Gradle, JDK, etc.) are never used, eliminating the risk of host-side tampering.

#### 🛡️ Isolation from Host

- The build process is fully contained. Even a compromised host cannot inject malicious code into the build.
- Globally installed packages or trojans on the host cannot affect what ends up in the final image.

#### 🔍 Scannable and Auditable

- Container images can be scanned for vulnerabilities at build time before the artifact is promoted.
- The exact Gradle and JDK version is pinned, making the build transparent and reproducible.

---

## 🚀 Runtime Stage

The runtime image is intentionally minimal — it contains only what is needed to run the application.

```dockerfile
FROM eclipse-temurin:25-jre-alpine

RUN addgroup -S spring && adduser -S spring -G spring -s /sbin/nologin

WORKDIR /app

COPY --from=build /app/build/libs/*.jar app.jar

RUN chown spring:spring /app/app.jar
RUN chmod 400 /app/app.jar

EXPOSE 8080

USER spring

ENTRYPOINT ["java", "-jar", "app.jar"]
```

### Security Measures

| Measure | Detail |
|---------|--------|
| 📦 **Minimal image** | JRE-only Alpine image — no compilers, no build tools, minimal attack surface |
| 👤 **Non-root user** | Application runs as `spring`, a system user with no password, no home directory, and no shell access (`/sbin/nologin`) |
| 🔒 **Read-only JAR** | `chmod 400` makes the application binary immutable at runtime |
| 📁 **Scoped working directory** | `/app` isolates application files and prevents accidental writes elsewhere |
| 🌐 **Single port exposed** | Only port `8080` is declared, reducing accidental exposure of unnecessary services |
| ⚡ **Direct execution** | `ENTRYPOINT` uses exec form — no wrapping shell, proper signal handling, clean shutdown |

---

## 🛠️ Usage

### Build the Docker Image

```bash
docker build -t spring-boot-app-java21-gradle .
```

Or using Make:

```bash
make build
```

### Run the Container

```bash
docker run -p 8080:8080 spring-boot-app-java21-gradle
```

Or using Make:

```bash
make run
```

### Call the API

```bash
curl http://localhost:8080/api/info
```

The service will be available at [http://localhost:8080](http://localhost:8080).
