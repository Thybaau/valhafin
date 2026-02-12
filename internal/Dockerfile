# Stage 1: Build
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files from parent directory
COPY ../go.mod ../go.sum ./

# Download dependencies
RUN go mod download

# Copy source code from parent directory
COPY ../ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o valhafin .

# Stage 2: Runtime
FROM alpine:3.23

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 valhafin && \
    adduser -D -u 1000 -G valhafin valhafin

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/valhafin .

# Change ownership
RUN chown -R valhafin:valhafin /app

# Switch to non-root user
USER valhafin

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./valhafin"]
