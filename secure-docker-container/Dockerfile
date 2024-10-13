###
# Build stage
###

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


###
# Runtime stage
###

# Create a lean production image with Java 21 JRE only
FROM eclipse-temurin:21-jre-alpine

# Create a non-root user 'spring' for security
RUN addgroup -S spring && adduser -S spring -G spring -s /sbin/nologin

# Set the working directory to /app
WORKDIR /app

# Copy the JAR file from the build stage to the production stage
COPY --from=build /app/build/libs/*.jar app.jar

# Change ownership of the JAR file to the 'spring' user and make it read-only
RUN chown spring:spring /app/app.jar
RUN chmod 400 /app/app.jar

# Expose the application port (e.g., 8080)
EXPOSE 8080

# Switch to the non-root 'spring' user
USER spring

# Use the entrypoint to start the Spring Boot application
ENTRYPOINT ["java", "-jar", "app.jar"]
