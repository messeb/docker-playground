# Secure Docker Container

This [Dockerfile](Dockerfile) is a multi-stage build that uses Java 21 for the build stage and Java 21 JRE for the runtime stage. It incorporates security best practices like non-root users, read-only files, and lean images.

### Build Stage

When using Docker for building applications, the **build stage** provides a secure and isolated environment, minimizing the risk of malicious dependencies and host contamination. Below is an explanation of the security benefits of using Docker in the build process:

```dockerfile
# Build the application using a Gradle image with Java 21
FROM gradle:8.10-jdk21 AS build

# Set the working directory in the container
WORKDIR /app

# Copy only the Gradle wrapper and dependencies initially to leverage Docker cache
COPY java-service/gradle/ ./gradle
COPY java-service/gradlew .
COPY java-service/build.gradle .
COPY java-service/settings.gradle .

# Download dependencies without building the project to improve build cache efficiency
RUN ./gradlew dependencies --no-daemon

# Copy the entire project and build it
COPY java-service .

# Build the application and create the fat/uber JAR
RUN ./gradlew bootJar --no-daemon
```


#### Controlled, Clean Build Environment

- **Fresh Environment Each Time**: Each time you build in a Docker container, you start with a fresh, known, and controlled environment. This ensures that any malicious or compromised dependencies from previous builds, system-wide tools, or misconfigurations aren't lingering in the environment.

- **No Host Contamination**: Building inside Docker ensures that the host system is not exposed to the build process. This means that any vulnerabilities or misconfigurations in your build tools (e.g., Gradle, JDK) cannot affect the host. The build process is fully isolated in a container, keeping the host secure.

- **Minimal Attack Surface**: In a container, you only install what's required for the build (e.g., Gradle, JDK). This reduces the likelihood of inadvertently including or using malicious software from the host system that may have been compromised.


#### Isolation from Host Compromises

- **Host Independence**: If the host machine has been compromised (e.g., with malicious software or trojans), building inside a container **can** isolates the build from those host threats, keeping the build process cleaner and more secure.

- **Avoiding Untrusted Global Dependencies**: On the host, globally installed packages, libraries, or build tools could have been tampered with or maliciously modified. Docker containers, with their own isolated environments, avoid reliance on these potentially untrusted host-side resources.


#### Easier to Scan and Monitor

- **Build-Time Security Scanning**: Containers make it easier to integrate security scanning tools (e.g., scanning the image for vulnerabilities or malicious packages) during the build. This ensures you can catch potentially malicious dependencies before they make it into the final application.

#### Reproducible and Transparent Builds

- **Known Sources**: Docker images (like the Gradle image) are typically pulled from trusted sources like Docker Hub, which are regularly vetted and maintained. You can control exactly which versions and tools you’re using, reducing the chance of malicious or outdated dependencies slipping in.

--- 

### Runtime Stage

The runtime stage of the Dockerfile uses a minimal JRE image and incorporates security best practices to ensure a secure and efficient runtime environment for the Spring Boot application:


#### Minimal and Lean Runtime Environment

```dockerfile
FROM eclipse-temurin:21-jre-alpine
```

- **Small Attack Surface**: Using a minimal **JRE (Java Runtime Environment)** image, like `eclipse-temurin:21-jre-alpine`, reduces the image size and eliminates unnecessary components like compilers or build tools. This minimizes the number of potential vulnerabilities in the production environment.


#### Non-Root User

```dockerfile
RUN addgroup -S spring && adduser -S spring -G spring -s /sbin/nologin
```

- **Running as Non-Root**: Creating and using a non-root user in the container ensures that the application does not have elevated privileges. Even if the application is compromised, the attacker’s access will be limited.

- **System User Advantage**: The `-S` flag in adduser creates a system user that has no password, no home directory, and limited login capabilities. This further restricts the actions that the user can perform, adding another layer of security.

- **Prevents shell access**: Using `/sbin/nologin` ensures that the user is unable to run any shell commands or interact with the system through a terminal, making it a common security practice for system users that should not need interactive access.

- **Security Advantage**: Running the application as the spring user instead of the root user helps prevent privilege escalation attacks. If the application is breached, the attacker won’t be able to make system-wide changes.


#### Setting the Working Directory

```dockerfile
WORKDIR /app
```

- **File Isolation**: By explicitly setting a working directory like `/app`, the application files (in this case, the JAR file) are placed in a known, controlled directory. This isolates them from other parts of the filesystem and minimizes the risk of accidental file access or modification from other parts of the container.

- **Prevents File Overwrite**: Any subsequent operations that involve writing or reading files are scoped to this directory, preventing files from being written or read from unsecured or unintended locations.


#### Ensures Non-Root Ownership

```dockerfile
RUN chown spring:spring /app/app.jar
```

- **Non-Root Execution**: Changing ownership to `spring:spring` makes sure that the application can run under the spring user without requiring elevated root privileges. Running applications as non-root limits the potential for exploitation and reduces the risk of privilege escalation attacks.


#### Read-Only JAR File

```dockerfile
RUN chmod 400 /app/app.jar
```

- **Immutable Application**: Making the JAR file read-only (`chmod 400`) ensures that the application code cannot be modified once the container is running. This provides immutability and ensures the application behaves exactly as deployed.


#### Port Configuration
```dockerfile
EXPOSE 8080
```

- **Controlled Network Exposure**: By specifying only the necessary port (8080 in this case), it helps to clarify that only the specific service on this port should be exposed. This reduces confusion and helps prevent accidental exposure of other ports that might be unnecessary and could increase the attack surface.


#### Switching to Non-Root User

```dockerfile
USER spring
```

- **Running as Non-Root**: Switching to the non-root user before starting the application ensures that the application process runs with minimal privileges. This practice limits the potential impact of security breaches by restricting the actions the application can perform.


#### Direct Execution of JAR File

```dockerfile
ENTRYPOINT ["java", "-jar", "app.jar"]
```

- **Direct Execution**: The Java process is executed directly, which improves container efficiency and ensures better signal handling.

- **No Extra Shell**: Not using the unnecessary shell commmand (`sh -c`) avoids potential issues with signal propagation and shutdown behavior, making the container more robust.


## Building the Docker Image

To build the Docker image, use the following command:

```bash 
docker build -t spring-boot-app-java21-gradle .
```

This command will create a Docker image with your Spring Boot application using Java 21 and Gradle.


## Running the Docker Container

To run the container, use the following command (or `make build`):

```bash
docker run -p 8080:8080 spring-boot-app-java21-gradle
```

This will start the Spring Boot application on port 8080. You can access it at http://localhost:8080.


## Fetch service endpoint 

Fetch the service endpoint using the following command (or `make run`):

```bash
curl http://localhost:8080/api/info
```
