// Package scheduler explores execution schedules deterministically.
//
// Same Scenario + same Config must always produce the same Result.
// There is no dependency on wall-clock time or the Go runtime scheduler.
package scheduler

import (
	"math/rand/v2"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Config controls exploration. Same Config + Scenario → same Result.
type Config struct {
	// Seed drives the deterministic scheduler. Zero is a valid seed.
	Seed uint64
	// MaxSteps caps how many transitions Explore may take (0 = unlimited).
	MaxSteps int
}

// Explore runs a deterministic schedule exploration for sc.
//
// At each step it picks among Enabled transitions using a seeded RNG.
// After every Apply it evaluates all invariants. On the first violation it
// returns a Result with Passed=false and a Witness carrying the schedule.
func Explore(sc scenario.Scenario, invariants []invariant.Invariant, cfg Config) witness.Result {
	rng := rand.New(rand.NewPCG(cfg.Seed, cfg.Seed^0x9e3779b97f4a7c15))
	state := sc.Initial()
	schedule := make(scenario.Schedule, 0, 16)

	for step := 0; cfg.MaxSteps == 0 || step < cfg.MaxSteps; step++ {
		enabled := sc.Enabled(state)
		if len(enabled) == 0 {
			return witness.Result{Passed: true, Steps: step, Schedule: schedule}
		}

		pick := enabled[rng.IntN(len(enabled))]
		state = sc.Apply(state, pick)
		schedule = append(schedule, pick)

		if w := invariant.Check(state, schedule, invariants); w != nil {
			return witness.Result{Passed: false, Steps: step + 1, Schedule: schedule, Witness: w}
		}
	}

	return witness.Result{Passed: true, Steps: cfg.MaxSteps, Schedule: schedule}
}
