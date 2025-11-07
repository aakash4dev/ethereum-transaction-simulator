# Build stage
FROM golang:1.20-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (needed for some Go dependencies)
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o simulator ./cmd/simulator

# Final stage - minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/simulator .

# Copy .env.example for reference (optional)
COPY --from=builder /app/.env.example .

# Make binary executable
RUN chmod +x simulator

# Run the simulator
CMD ["./simulator"]

