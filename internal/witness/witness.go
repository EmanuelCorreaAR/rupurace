// Package witness holds failure evidence and exploration results.
package witness

import "github.com/EmanuelCorreaAR/rupurace/internal/scenario"

// Witness captures the evidence required to reproduce a failure.
type Witness struct {
	// Schedule is the exact transition sequence that led to the violation.
	Schedule scenario.Schedule
	// Invariant is the name of the property that failed (e.g. "balance >= 0").
	Invariant string
	// Violation is the detail message from the failed check.
	Violation string
	// FailedAt is the 1-based schedule length at which the invariant failed.
	FailedAt int
}

// Result is the outcome of Explore or Replay.
type Result struct {
	Passed   bool
	Steps    int
	Schedule scenario.Schedule
	Witness  *Witness
	Explored int // schedules fully explored (exhaustive); 0 if unused
	Strategy string
}
