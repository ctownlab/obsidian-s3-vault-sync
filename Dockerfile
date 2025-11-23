# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o vault-sync ./cmd

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/vault-sync .

# Create directories for vaults and tarballs
RUN mkdir -p /data/vaults /data/tarballs

# Set default environment variables
ENV VAULT_DIR=/data/vaults
ENV TAR_DIR=/data/tarballs

ENTRYPOINT ["./vault-sync"]
CMD ["run", "--help"]

