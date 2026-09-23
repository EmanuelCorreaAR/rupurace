// Package invariant checks state after each transition.
package invariant

import (
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Invariant is checked after every transition. A non-nil error is a violation.
type Invariant func(scenario.State) error

// Check evaluates all invariants against state. On the first failure it returns
// a Witness carrying the schedule so far; otherwise nil.
func Check(state scenario.State, schedule scenario.Schedule, invariants []Invariant) *witness.Witness {
	for _, inv := range invariants {
		if inv == nil {
			continue
		}
		if err := inv(state); err != nil {
			schedCopy := make(scenario.Schedule, len(schedule))
			copy(schedCopy, schedule)
			return &witness.Witness{
				Schedule:  schedCopy,
				Violation: err.Error(),
			}
		}
	}
	return nil
}
