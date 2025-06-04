# Start from the official Golang image for building
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install git (required for go mod)
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go app (static binary)
RUN CGO_ENABLED=0 GOOS=linux go build -o /slack-payment-bot .

# Use a minimal base image for running
FROM alpine:latest
WORKDIR /root/

# Copy the built binary from builder
COPY --from=builder /slack-payment-bot .

# Expose the port the app runs on
EXPOSE 8080

# Set environment variables (can be overridden at runtime)
ENV PORT=8080

# Command to run
ENTRYPOINT ["./slack-payment-bot"] 