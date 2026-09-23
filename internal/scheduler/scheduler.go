// Package scheduler explores execution schedules deterministically.
//
// Same Scenario + same Config must always produce the same Result.
// There is no dependency on wall-clock time or the Go runtime scheduler.
package scheduler

import (
	"cmp"
	"math/rand/v2"
	"slices"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Config controls seeded exploration.
type Config struct {
	// Seed drives the deterministic scheduler. Zero is a valid seed.
	Seed uint64
	// MaxSteps caps how many transitions Explore may take (0 = unlimited).
	MaxSteps int
}

// ExhaustiveConfig controls exhaustive schedule search.
type ExhaustiveConfig struct {
	// MaxDepth caps schedule length (0 = unlimited).
	MaxDepth int
}

// Explore runs a seeded schedule exploration for sc.
func Explore(sc scenario.Scenario, specs []invariant.Spec, cfg Config) witness.Result {
	rng := rand.New(rand.NewPCG(cfg.Seed, cfg.Seed^0x9e3779b97f4a7c15))
	state := sc.Initial()
	schedule := make(scenario.Schedule, 0, 16)

	for step := 0; cfg.MaxSteps == 0 || step < cfg.MaxSteps; step++ {
		enabled := sortedEnabled(sc.Enabled(state))
		if len(enabled) == 0 {
			return witness.Result{Passed: true, Steps: step, Schedule: schedule, Strategy: "seeded"}
		}

		pick := enabled[rng.IntN(len(enabled))]
		state = sc.Apply(state, pick)
		schedule = append(schedule, pick)

		if w := invariant.Check(state, schedule, specs); w != nil {
			return witness.Result{Passed: false, Steps: step + 1, Schedule: schedule, Witness: w, Strategy: "seeded"}
		}
	}

	return witness.Result{Passed: true, Steps: cfg.MaxSteps, Schedule: schedule, Strategy: "seeded"}
}

// ExploreExhaustive DFS-searches all schedules in a stable order.
// The first violating schedule (by that order) is returned as the Witness.
// If every schedule passes, Result.Passed is true.
func ExploreExhaustive(sc scenario.Scenario, specs []invariant.Spec, cfg ExhaustiveConfig) witness.Result {
	type frame struct {
		state    scenario.State
		schedule scenario.Schedule
	}

	stack := []frame{{state: sc.Initial(), schedule: nil}}
	explored := 0

	for len(stack) > 0 {
		n := len(stack) - 1
		fr := stack[n]
		stack = stack[:n]

		if cfg.MaxDepth > 0 && len(fr.schedule) >= cfg.MaxDepth {
			explored++
			continue
		}

		enabled := sortedEnabled(sc.Enabled(fr.state))
		if len(enabled) == 0 {
			explored++
			continue
		}

		// Push in reverse so the lowest-ordered transition is tried first (DFS).
		for i := len(enabled) - 1; i >= 0; i-- {
			t := enabled[i]
			nextState := sc.Apply(fr.state, t)
			nextSched := append(fr.schedule[:len(fr.schedule):len(fr.schedule)], t)

			if w := invariant.Check(nextState, nextSched, specs); w != nil {
				return witness.Result{
					Passed:   false,
					Steps:    len(nextSched),
					Schedule: nextSched,
					Witness:  w,
					Explored: explored,
					Strategy: "exhaustive",
				}
			}

			stack = append(stack, frame{state: nextState, schedule: nextSched})
		}
	}

	return witness.Result{Passed: true, Explored: explored, Strategy: "exhaustive"}
}

func sortedEnabled(enabled []scenario.Transition) []scenario.Transition {
	out := slices.Clone(enabled)
	slices.SortFunc(out, func(a, b scenario.Transition) int {
		if c := cmp.Compare(string(a.Worker), string(b.Worker)); c != 0 {
			return c
		}
		return cmp.Compare(a.Step, b.Step)
	})
	return out
}
