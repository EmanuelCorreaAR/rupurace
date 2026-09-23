// Package witness holds failure evidence and exploration results.
package witness

import "github.com/EmanuelCorreaAR/rupurace/internal/scenario"

// Witness captures the evidence required to reproduce a failure.
type Witness struct {
	// Schedule is the exact transition sequence that led to the violation.
	Schedule scenario.Schedule
	// Violation is the invariant error message.
	Violation string
}

// Result is the outcome of Explore or Replay.
type Result struct {
	Passed   bool
	Steps    int
	Schedule scenario.Schedule
	Witness  *Witness
}
