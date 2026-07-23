# Contributing to Cache Local

## Development Setup

### Prerequisites

- Go 1.26.x
- golangci-lint (optional but recommended)

### Clone and build

```bash
git clone https://github.com/Muxcore-Media/cache-local.git
cd cache-local
make build
```

### Run against a local muxcored

```bash
# Terminal 1: start core in dev mode
cd ../core
MUXCORE_INSECURE_DISABLE_TLS=true ./muxcored

# Terminal 2: start module
make build && ./cache-local --muxcore-mesh-addr localhost:9090
```

## Running Tests

```bash
make test
```

Tests must not depend on a running muxcored instance. Use mocks where needed.

## Linting

```bash
make lint
```

## Pull Requests

- Keep changes focused
- Include tests for behavior changes
- Run `make test` before opening a PR
