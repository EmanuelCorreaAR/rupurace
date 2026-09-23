# Changelog

## [0.2.1] - 2026-09-23

### Added
- Safe exploration-node pruning (`--prune`): `NodeKey = state fingerprint + enabled set`
- `EquivalentPruned > 0` when re-expansion is skipped
- Tests: prune reduces schedules/states on `interleave`; canonical first witness unpruned≡pruned; pruned witness replays; NodeKey adversarial (same state FP ≠ collapse when enabled differs)

### Changed
- Default remains unpruned (0.2.0 baseline). Pruning is opt-in so measurement stays honest.

## [0.2.0] - 2026-09-23

### Added
- Exploration `Stats`: possible/executed schedules, states visited/unique, violations, duration
- Synthetic scenario `interleave` (W workers × S steps) to confront combinatorial explosion
- Flags: `--stats`, `--continue`, `--measure-memory`, `--workers`, `--steps`
- Multinomial `PossibleSchedules` for fixed per-worker step counts
- Measurement tests: 2×5 = 252 full explorations; stats determinism; continue counts violations

### Changed
- Exhaustive default path unchanged (stop on first witness) — constitutional equation protected
- Still no public Go API; EquivalentPruned stays 0 until hashing actually prunes

## [0.1.2] - 2026-09-23

### Added
- Constitutional test: explore → witness → replay preserves invariant, failedAt, schedule
- Witness JSON round-trip (`witness_v1`): scenario / schedule / invariant / failedAt
- Direct package tests: `invariant`, `replay`, `report`, `scenario`, `witness`

### Changed
- README identity vs `go test -race`: conflictive memory access vs orderings that violate a property
- Local `witness.json` artifacts ignored by git

## [0.1.1] - 2026-09-23

### Added
- Scenarios that stress the cooperative model: `lost-update`, `check-then-act`, `init-ordering`
- Aggressive hardening tests: exhaustive determinism ×50, replay identity ×50, scrambled `Enabled`, Apply immutability
- CLI coverage: explore → witness → replay for every failing built-in scenario

### Changed
- README lists the scenario battery; still no public Go API

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
