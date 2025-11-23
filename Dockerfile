# Multi-stage build for optimized image
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    gcc \
    g++ \
    musl-dev \
    ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vision-api cmd/app/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/vision-api .
COPY --from=builder /app/config.yaml .

# Copy web templates and static files
COPY --from=builder /app/web ./web

# Expose port
EXPOSE 8080

ENTRYPOINT ["./vision-api"]
