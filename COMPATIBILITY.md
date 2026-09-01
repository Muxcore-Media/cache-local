# Compatibility

## Core Version

| Module Version | Core Version | Status |
|----------------|-------------|--------|
| v0.1.1         | v0.5.8+     | Current |

## Contracts

| Contract | Capability | Status |
|----------|-----------|--------|
| CacheLayer | `cache.local` | Current (canonical) |
| CacheLayer | `cache.memory` | Legacy alias (same implementation) |

## `cache.local` vs `cache.memory`

| Capability | Meaning |
|------------|---------|
| **`cache.local`** | Canonical MuxCore capability for a **process-local, in-memory** `CacheLayerService` used as a storage read-through cache. This is what core documents (`CapabilityCacheLocal`). |
| **`cache.memory`** | **Legacy alias** advertised for compatibility with the retired `cache-memory` dump / older dials. Same process, same map, same TTL sweeper — not a separate backend. |

Do **not** run `cache-local` alongside another module that also claims `cache.local` / `cache.memory`. Prefer `cache-redis` when you need a shared cache across hosts.

Data is **not durable**: restart clears the map. Single-node only.

Default listen address is **`:9602`** (`CACHE_LOCAL_GRPC_ADDR`). Legacy installers may still reference `:9600` (the `cache-redis` port).

## Breaking Changes

This is a pre-1.0 module. Interfaces may change without notice.
