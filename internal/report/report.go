// Package report serializes schedules and exploration results for JSON I/O.
package report

import (
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/version"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Step is one schedule entry in JSON form.
type Step struct {
	Worker string `json:"worker"`
	Step   int    `json:"step"`
}

// ResultView is the audit result payload for explore/replay.
type ResultView struct {
	Passed   bool   `json:"passed"`
	Steps    int    `json:"steps"`
	Strategy string `json:"strategy,omitempty"`
	Explored int    `json:"explored,omitempty"`
	Schedule []Step `json:"schedule"`
	Witness  *File  `json:"witness,omitempty"`
}

// File is the portable witness artifact for `rupurace replay`.
//
// Shape (conceptual heart of RupuRace):
//
//	{
//	  "scenario": "double-withdraw",
//	  "schedule": [{"worker":"A","step":0}, ...],
//	  "invariant": "balance >= 0",
//	  "failedAt": 4
//	}
type File struct {
	Tool      string         `json:"tool,omitempty"`
	Version   string         `json:"version,omitempty"`
	Kind      string         `json:"kind,omitempty"`
	Scenario  string         `json:"scenario"`
	Params    map[string]any `json:"params,omitempty"`
	Schedule  []Step         `json:"schedule"`
	Invariant string         `json:"invariant"`
	FailedAt  int            `json:"failedAt"`
	Violation string         `json:"violation,omitempty"`
}

const WitnessKind = "witness_v1"

// FromSchedule converts an engine schedule to JSON steps.
func FromSchedule(s scenario.Schedule) []Step {
	out := make([]Step, len(s))
	for i, t := range s {
		out[i] = Step{Worker: string(t.Worker), Step: t.Step}
	}
	return out
}

// ToSchedule converts JSON steps back to an engine schedule.
func ToSchedule(steps []Step) scenario.Schedule {
	out := make(scenario.Schedule, len(steps))
	for i, s := range steps {
		out[i] = scenario.Transition{
			Worker: scenario.WorkerID(s.Worker),
			Step:   s.Step,
		}
	}
	return out
}

// FromResult maps an engine Result to the audit result view.
func FromResult(r witness.Result, scenarioName string, params map[string]any) ResultView {
	view := ResultView{
		Passed:   r.Passed,
		Steps:    r.Steps,
		Strategy: r.Strategy,
		Explored: r.Explored,
		Schedule: FromSchedule(r.Schedule),
	}
	if r.Witness != nil {
		f := NewWitnessFile(scenarioName, params, *r.Witness)
		view.Witness = &f
	}
	return view
}

// NewWitnessFile builds a portable witness artifact.
func NewWitnessFile(scenarioName string, params map[string]any, w witness.Witness) File {
	return File{
		Tool:      "rupurace",
		Version:   version.Version,
		Kind:      WitnessKind,
		Scenario:  scenarioName,
		Params:    params,
		Schedule:  FromSchedule(w.Schedule),
		Invariant: w.Invariant,
		FailedAt:  w.FailedAt,
		Violation: w.Violation,
	}
}
