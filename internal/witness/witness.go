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

// Stats summarizes an exploration run. Used to confront combinatorial explosion
// before implementing pruning. EquivalentPruned stays 0 until state hashing
// actually cuts the search.
type Stats struct {
	Workers           int    `json:"workers"`
	Transitions       int    `json:"transitions"` // steps in a full interleaving (sum of per-worker steps)
	PossibleSchedules uint64 `json:"possibleSchedules,omitempty"`
	SchedulesExplored int    `json:"schedulesExplored"`
	StatesVisited     int    `json:"statesVisited"`
	UniqueStates      int    `json:"uniqueStates"`
	Violations        int    `json:"violations"`
	EquivalentPruned  int    `json:"equivalentPruned"`
	DurationNanos     int64  `json:"durationNanos"`
	HeapBytes         uint64 `json:"heapBytes,omitempty"`
}

// Result is the outcome of Explore or Replay.
type Result struct {
	Passed   bool
	Steps    int
	Schedule scenario.Schedule
	Witness  *Witness
	Explored int // schedules fully explored (exhaustive); 0 if unused
	Strategy string
	Stats    *Stats
}
