// Package scenario defines abstract concurrent programs under test.
//
// A Scenario exposes enabled transitions from a State; Apply advances state.
// The core does not control real goroutines yet.
package scenario

import "fmt"

// ActorID names a concurrent participant in a scenario.
type ActorID string

// StepID names a discrete step an actor may take.
type StepID string

// Transition is one scheduled step: which actor runs which step.
type Transition struct {
	Actor ActorID
	Step  StepID
}

func (t Transition) String() string {
	return fmt.Sprintf("%s:%s", t.Actor, t.Step)
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
