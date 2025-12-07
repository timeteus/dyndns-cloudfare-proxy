# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dyndns-cloudflare-proxy .

# Final stage - using scratch for minimal image size
FROM scratch

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/dyndns-cloudflare-proxy .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./dyndns-cloudflare-proxy"]
