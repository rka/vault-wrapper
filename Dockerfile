# Build stage
FROM golang:1.21-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git build-base

WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO disabled for better compatibility
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:3.18

# Add necessary runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary and static files from builder
COPY --from=builder /app/main .
COPY --from=builder /app/static ./static

# Set necessary permissions
RUN chmod +x /app/main

EXPOSE 3001

CMD ["./main"]
