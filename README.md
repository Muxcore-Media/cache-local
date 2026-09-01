# Cache Local

[![CI](https://git.zem.systems/muxcore/cache-local/actions/workflows/ci.yml/badge.svg)](https://git.zem.systems/muxcore/cache-local/actions)
[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://go.dev/)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

**In-memory local read-through cache for the MuxCore storage orchestrator.**

A MuxCore sidecar module that implements the `cache.local` capability via
`CacheLayerService` (Get / Set / Invalidate). Entries expire after a default TTL;
a background sweeper reclaims expired keys.

---

## How It Works

```
storage orchestrator ──gRPC──→ cache-local HTTPAddr (:9602) ──→ in-memory map
                              (CacheLayerService via pkg/client)
```

Core discovers the sidecar by capability (`cache.local`) and dials the module's
gRPC address from registration (`HTTPAddr`). Use `github.com/Muxcore-Media/cache-local/pkg/client`
for a `contracts.CacheLayer` adapter over `CacheLayerService`.

No external store. Data is process-local and lost on restart.

> **Port note:** This module listens on **`:9602`** by default (`CACHE_LOCAL_GRPC_ADDR`).
> Do not confuse with **`cache-redis`** (`:9600`) or older installers that defaulted `:9600`.

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `CACHE_LOCAL_GRPC_ADDR` | `:9602` | gRPC listen address |
| `CACHE_LOCAL_TTL` | `5m` | Default TTL for new entries (Go duration) |
| `CACHE_LOCAL_MAX_BYTES` | `268435456` (256 MiB) | Total in-memory cache size cap |
| `CACHE_LOCAL_MAX_ENTRY_BYTES` | `33554432` (32 MiB) | Per-entry size cap (matches core default gRPC limit) |
| `CACHE_LOCAL_MAX_ENTRIES` | `0` (unlimited) | Optional key count cap |
| `MUXCORE_GRPC_ADDR` | — | Core mesh address (or `--muxcore-mesh-addr`) |
| `MUXCORE_MODULE_ID` | `cache-local` | Module ID override (or `--muxcore-module-id`) |

Per-entry max on the wire is **32 MiB** (`server.MaxValueBytes`); oversize `Set` returns `InvalidArgument`.

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
