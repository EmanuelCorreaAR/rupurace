// Package audit builds the Rupu-family deterministic JSON envelope:
// input → configuration → method → result.
package audit

import "github.com/EmanuelCorreaAR/rupurace/internal/version"

const (
	Tool   = "rupurace"
	Family = "rupu"
)

// Method tags the algorithm versions used for this report.
var Method = map[string]string{
	"unit":       "abstract_scenario",
	"explore":    "exhaustive_scheduler_v1",
	"seeded":     "seeded_scheduler_v1",
	"replay":     "schedule_replay_v1",
	"invariant":  "post_transition_v1",
	"minimize":   "one_minimality_v1",
	"transition": "cooperative_step_v1",
}

// Envelope is the top-level audit report.
type Envelope struct {
	Tool          string            `json:"tool"`
	Version       string            `json:"version"`
	Family        string            `json:"family"`
	Command       string            `json:"command"`
	Input         map[string]any    `json:"input"`
	Configuration map[string]any    `json:"configuration"`
	Method        map[string]string `json:"method"`
	Result        any               `json:"result"`
}

// Build returns a frozen-shape envelope for command.
func Build(command string, input, configuration map[string]any, result any) Envelope {
	if input == nil {
		input = map[string]any{}
	}
	if configuration == nil {
		configuration = map[string]any{}
	}
	return Envelope{
		Tool:          Tool,
		Version:       version.Version,
		Family:        Family,
		Command:       command,
		Input:         input,
		Configuration: configuration,
		Method:        Method,
		Result:        result,
	}
}
