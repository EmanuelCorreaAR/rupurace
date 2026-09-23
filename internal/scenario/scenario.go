// Package scenario defines abstract concurrent programs under test.
//
// Pipeline:
//
//	Scenario → possible transitions → schedule explorer → state transition
//	  → invariant check → (OK continue | FAIL → witness → replay)
//
// Transitions are cooperative and explicit: RupuRace schedules which worker
// runs which step index. It does not depend on the Go runtime scheduler.
package scenario

import "fmt"

// WorkerID names a concurrent participant in a scenario.
type WorkerID string

// Transition is one scheduled step: which worker runs which step index.
type Transition struct {
	Worker WorkerID
	Step   int
}

func (t Transition) String() string {
	return fmt.Sprintf("%s:%d", t.Worker, t.Step)
}

// Schedule is an ordered sequence of transitions (a witnessable execution).
type Schedule []Transition

// State is the scenario state after zero or more transitions.
// Implementations must be immutable or treat Apply as returning a new value.
type State any

// Scenario defines an abstract concurrent program under test.
type Scenario interface {
	// Initial returns the starting state.
	Initial() State
	// Enabled returns the transitions that may run from s.
	Enabled(s State) []Transition
	// Apply returns the state after executing t. t must be enabled in s.
	Apply(s State, t Transition) State
}
