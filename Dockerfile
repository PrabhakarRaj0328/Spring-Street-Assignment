# Build Stage
FROM golang:1.21-alpine AS builder

# Set the working directory
WORKDIR /app

# Install required system dependencies (for building some Go tools if needed)
RUN apk add --no-cache git ca-certificates

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies.
RUN go mod download

# Copy the source code
COPY . .

# Build the API binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

# Build the ETL binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /etl ./cmd/etl

# Final Production Stage
FROM alpine:latest

# Install CA certificates for HTTPS requests (Yahoo Finance)
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

# Copy the pre-built binary files from the builder stage
COPY --from=builder /api .
COPY --from=builder /etl .

# Expose API port
EXPOSE 8080

# The command is overridden in docker-compose.yml
CMD ["./api"]
