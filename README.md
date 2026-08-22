# Cache Local

[![CI](https://git.zem.systems/muxcore/cache-local/actions/workflows/ci.yml/badge.svg)](https://github.com/Muxcore-Media/cache-local/actions)
[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

**In-memory local read-through cache for the MuxCore storage orchestrator.**

A MuxCore sidecar module that implements the `cache.local` capability via
`CacheLayerService` (Get / Set / Invalidate). Entries expire after a default TTL;
a background sweeper reclaims expired keys.

---

## How It Works

```
storage orchestrator ──→ DialCacheLayer ──→ cache-local (gRPC) ──→ in-memory map
```

No external store. Data is process-local and lost on restart.

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CACHE_LOCAL_GRPC_ADDR` | `:9602` | gRPC listen address |
| `MUXCORE_GRPC_ADDR` | — | Core mesh address (or `--muxcore-mesh-addr`) |
| `MUXCORE_MODULE_ID` | `cache-local` | Module ID override (or `--muxcore-module-id`) |

---

## Quick Start

```bash
make build
./cache-local --muxcore-mesh-addr localhost:9090
```

Core must be reachable (dev: `MUXCORE_INSECURE_DISABLE_TLS=true ./muxcored` in `../core`).

---

## Capability

| Capability | Role |
|------------|------|
| `cache.local` | **Canonical** — process-local in-memory `CacheLayerService` |
| `cache.memory` | **Legacy alias** for the retired `cache-memory` dump; same implementation |

See [COMPATIBILITY.md](COMPATIBILITY.md). Prefer [`cache-redis`](https://github.com/Muxcore-Media/cache-redis) for multi-host shared cache.

## License

GPL-3.0
