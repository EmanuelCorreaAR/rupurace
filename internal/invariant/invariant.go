// Package invariant checks named properties after each transition.
package invariant

import (
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Spec is a named invariant checked after every transition.
type Spec struct {
	Name string
	Hold func(scenario.State) error
}

// Check evaluates all specs against state. On the first failure it returns
// a Witness carrying the schedule so far; otherwise nil.
func Check(state scenario.State, schedule scenario.Schedule, specs []Spec) *witness.Witness {
	for _, spec := range specs {
		if spec.Hold == nil {
			continue
		}
		if err := spec.Hold(state); err != nil {
			schedCopy := make(scenario.Schedule, len(schedule))
			copy(schedCopy, schedule)
			name := spec.Name
			if name == "" {
				name = err.Error()
			}
			return &witness.Witness{
				Schedule:  schedCopy,
				Invariant: name,
				Violation: err.Error(),
				FailedAt:  len(schedCopy),
			}
		}
	}
	return nil
}
