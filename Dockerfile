# Build stage
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary using native cross-compilation
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o vault-sync ./cmd

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from builder to /usr/local/bin
COPY --from=builder /app/vault-sync /usr/local/bin/vault-sync

# Make binary executable
RUN chmod +x /usr/local/bin/vault-sync

# Set default environment variables
ENV VAULT_DIR=/data/vaults
ENV TAR_DIR=/data/tarballs

# Set workdir but don't create /data (let volume mount handle it)
WORKDIR /data

ENTRYPOINT ["/usr/local/bin/vault-sync"]
CMD ["run", "--help"]

