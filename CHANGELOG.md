# Changelog

## [0.0.1] - 2026-09-23
### Added
- Project scaffolding: README, Apache-2.0, single Go module at repo root
- Layout: `cmd/rupurace` stub + `internal/{scheduler,scenario,invariant,witness,replay}`
- Minimal deterministic `Explore` / `Replay` over abstract scenarios
- Determinism tests: same scenario + same configuration → same result
- Version via `internal/version` (no public library package yet)
