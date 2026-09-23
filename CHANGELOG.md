# Changelog

## [0.1.0] - 2026-09-23

### Added
- Cooperative transition model: `{worker, step}` — no Go runtime scheduler
- Built-in scenario `double-withdraw` (2 workers × 3 steps, shared balance)
- Exhaustive schedule exploration (`--strategy exhaustive`)
- Witness artifact with `scenario`, `schedule`, `invariant`, `failedAt`
- Heart property: explore finds bug → write witness → replay fails at the same transition every time

### Changed
- Default scenario is `double-withdraw`; default strategy is `exhaustive`
- `-o` writes `witness.json` directly (portable replay input)
- Positioning: systematic schedule exploration with deterministic failure replay

## [0.0.2] - 2026-09-23

### Added
- CLI `explore` / `replay` over the built-in abstract `counter` scenario
- Deterministic JSON audit envelope (`tool/version/family/command/input/configuration/method/result`)
- Portable `witness_v1` artifact for replay (`*.witness.json`)
- Flags: `--seed`, `--max-steps`, `--max-value`, `--fail-on-violation`, `--json`, `-o`
- Exit codes aligned with the Rupu family (`0` ok, `1` usage/I/O, `2` gate)

## [0.0.1] - 2026-09-23

### Added
- Project scaffolding: README, Apache-2.0, single Go module at repo root
- Layout: `cmd/rupurace` + `internal/{scheduler,scenario,invariant,witness,replay}`
- Minimal deterministic `Explore` / `Replay` over abstract scenarios
- Determinism tests: same scenario + same configuration → same result
- Version via `internal/version` (no public library package yet)
