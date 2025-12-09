# Build stage
FROM golang:1.25-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git build-base

WORKDIR /app

# Add version argument
ARG VERSION=unknown

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with version information
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=${VERSION}" -o main .

# Final stage
FROM alpine:3.18

# Add version label
ARG VERSION=unknown
LABEL version="${VERSION}"

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
