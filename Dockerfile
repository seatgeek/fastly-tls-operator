# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files (if they exist)
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fastly-operator ./cmd

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/fastly-operator .

# Expose port if needed (adjust as necessary)
# EXPOSE 8080

# Run the binary
CMD ["./fastly-operator"] 