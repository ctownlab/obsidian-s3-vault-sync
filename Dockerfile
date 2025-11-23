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

