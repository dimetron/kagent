# Build stage
FROM golang:1.25.1-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go/go.mod go/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY go/ ./

# Build the executor
RUN CGO_ENABLED=0 GOOS=linux go build -o temporal-executor ./cmd/temporal-executor

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /build/temporal-executor .

# Copy default configuration
COPY config/temporal-executor.yaml /app/config/

# Expose HTTP port
EXPOSE 8080

# Run the executor
ENTRYPOINT ["/app/temporal-executor"]
CMD ["/app/config/temporal-executor.yaml"]
