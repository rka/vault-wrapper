# Build stage
FROM golang:1.26-alpine AS builder

# Install necessary build tools
RUN apk add --no-cache git

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
FROM alpine:3.24

# Add version label
ARG VERSION=unknown
LABEL version="${VERSION}"

# Add necessary runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

# Copy binary and static files from builder
COPY --from=builder --chown=app:app /app/main .
COPY --from=builder --chown=app:app /app/static ./static

# Set necessary permissions
RUN chmod +x /app/main

USER app

EXPOSE 3001

CMD ["./main"]
