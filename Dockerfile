# Stage 1: Build both Go applications
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the main Gin API
RUN CGO_ENABLED=0 go build 

# Build the redis helper by changing to the redis directory
WORKDIR /app/redis
RUN CGO_ENABLED=0 go build -o /redis-helper .
RUN ls -l /redis-helper

# Stage 2: Final image
FROM alpine:latest

WORKDIR /app

# Copy binaries
COPY --from=builder /app/material-api /app/material-api
COPY --from=builder /app/redis/redis-helper /app/redis-helper

# Copy env
COPY .env /app/.env

# Run both binaries
CMD ["/bin/sh", "-c", "/app/material-api & /app/redis-helper"]