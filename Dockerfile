# Step 1: Use official Golang image to build the app
FROM golang:1.24 AS builder

# Set working directory inside container
WORKDIR /app

# Copy go.mod and go.sum first (for dependency caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary
RUN go build -o currency-watcher-backend ./main.go

# Step 2: Use a lightweight image for running the app
FROM alpine:latest

# Install certificates (needed for HTTPS requests)
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/currency-watcher-backend .

# Expose the port your app runs on
EXPOSE 8080

# Command to run the backend
CMD ["./currency-watcher-backend"]
