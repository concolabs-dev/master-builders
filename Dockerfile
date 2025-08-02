# Stage 1: Build the Go application
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o /master-builders-app .

# Stage 2: Create the final lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the built application from the builder stage
COPY --from=builder /master-builders-app /app/master-builders-app

# Copy the .env file
COPY .env /app/.env

# Copy templates to the correct location
COPY --from=builder /app/email/templates /app/templates

# Command to run the application
CMD ["/app/master-builders-app"]