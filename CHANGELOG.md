# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0/).

## [0.1.2] - 2026-10-05

### Changed
- CI runs on GitHub-hosted runners from the umbrella template; retired-origin workflows removed.
- Dependencies resolve from published GitHub tags (no filesystem `replace`); requires core v0.6.0.

## [0.1.1] — 2026-08-10

### Added

- SettingsProvider for `default_ttl` (`CACHE_LOCAL_TTL`) with live update
- Advertises `settings` capability for admin-ui discovery

## [0.1.0] — 2026-08-09

### Added

- Initial in-memory `CacheLayerService` sidecar (`cache.local`).
- Legacy `cache.memory` capability alias for compatibility with the retired `cache-memory` dump (same implementation).
- COMPATIBILITY notes clarifying `cache.local` vs `cache.memory`.
