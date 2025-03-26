# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application with optimized flags
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -gcflags="-l=4" \
    -o treemap ./cmd/treemap && \
    ls -l treemap && \
    chmod +x treemap

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder and make it executable
COPY --from=builder /app/treemap /app/treemap
RUN chmod +x /app/treemap && \
    ls -l /app/treemap

# Set the entrypoint
ENTRYPOINT ["/app/treemap"] 