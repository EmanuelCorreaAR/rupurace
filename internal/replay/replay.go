// Package replay re-executes a Witness schedule deterministically.
package replay

import (
	"fmt"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Run re-executes w.Schedule against sc and re-checks invariants.
// It does not consult an RNG: the schedule is the sole source of ordering.
func Run(sc scenario.Scenario, specs []invariant.Spec, w witness.Witness) (witness.Result, error) {
	state := sc.Initial()
	schedule := make(scenario.Schedule, 0, len(w.Schedule))

	for i, t := range w.Schedule {
		enabled := sc.Enabled(state)
		if !containsTransition(enabled, t) {
			return witness.Result{}, fmt.Errorf("replay: step %d transition %s not enabled", i, t)
		}
		state = sc.Apply(state, t)
		schedule = append(schedule, t)

		if got := invariant.Check(state, schedule, specs); got != nil {
			return witness.Result{
				Passed:   false,
				Steps:    i + 1,
				Schedule: schedule,
				Witness:  got,
				Strategy: "replay",
			}, nil
		}
	}

	return witness.Result{Passed: true, Steps: len(schedule), Schedule: schedule, Strategy: "replay"}, nil
}

func containsTransition(enabled []scenario.Transition, want scenario.Transition) bool {
	for _, t := range enabled {
		if t == want {
			return true
		}
	}
	return false
}
