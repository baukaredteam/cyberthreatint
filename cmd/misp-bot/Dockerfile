# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install required packages
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o misp-bot ./cmd/misp-bot

# Final stage
FROM alpine:3.18

WORKDIR /app

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/misp-bot .

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Run the application
CMD ["./misp-bot"]
