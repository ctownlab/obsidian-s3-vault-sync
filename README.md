# obsidian-s3-vault-sync
Sync Obsidian vault from S3 locally and manage tar backups

Built with [Cobra](https://github.com/spf13/cobra) for a powerful CLI experience.

## Quick Start

### Build and run
```bash
# Build the binary
go build -o vault-sync ./cmd/vault-sync

# Run it
./vault-sync run

# Get help
./vault-sync --help
./vault-sync run --help
```

### Run directly without building
```bash
go run ./cmd/vault-sync run
```

### Install globally
```bash
go install ./cmd/vault-sync
vault-sync run
```

## Usage

```bash
# Run the vault sync
vault-sync run

# Show help for any command
vault-sync --help
vault-sync run --help
```

## Development

```bash
# Run tests
go test ./...

# Build for current platform
go build -o vault-sync ./cmd/vault-sync

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o vault-sync-linux-amd64 ./cmd/vault-sync
GOOS=darwin GOARCH=arm64 go build -o vault-sync-darwin-arm64 ./cmd/vault-sync
```
