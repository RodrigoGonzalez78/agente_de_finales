# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies (needed for go-sqlite3 which requires CGO)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app (Updated path)
RUN CGO_ENABLED=1 GOOS=linux go build -o main cmd/server/main.go

# Run stage
FROM alpine:latest

WORKDIR /app

# Install simple dependencies
RUN apk add --no-cache ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy templates
COPY --from=builder /app/templates ./templates

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
