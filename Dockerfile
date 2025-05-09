# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
# Ensure your main package is correctly referenced, e.g., ./cmd/server/main.go or just ./ if main.go is in the root
RUN CGO_ENABLED=0 go build -o /master-builders-app .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /master-builders-app /app/master-builders-app

# Copy the .env file into the working directory of the application.
# The Go app uses godotenv.Load() and will look for .env here.
# Environment variables set in docker-compose.yml will override these if godotenv doesn't overwrite.
COPY .env /app/.env

# The application will listen on the PORT specified in the .env file (e.g., 8040)
# EXPOSE 8040 (This is metadata; actual port mapping is in docker-compose.yml)

# Command to run the application
CMD ["/app/master-builders-app"]
