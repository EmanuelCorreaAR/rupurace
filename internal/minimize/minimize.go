// Package minimize derives a shorter witness that still reproduces the same
// semantic violation. failedAt and schedule may change; that is expected.
//
// Strategy: 1-minimality (delta debugging). No single transition can be removed
// while still reproducing the same semantic violation. Minimize is deterministic.
//
// Invalid trials (illegal schedules) are not removable — they are required
// structure of the scenario, not "noise".
package minimize

import (
	"fmt"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// SemanticViolation is the identity of a failure for minimization purposes.
// Schedule length and failedAt are intentionally excluded.
type SemanticViolation struct {
	Invariant string
}

// OfWitness extracts the semantic violation identity from a witness.
func OfWitness(w witness.Witness) SemanticViolation {
	return SemanticViolation{Invariant: w.Invariant}
}

// Same reports whether two failures are the same semantic violation.
func (v SemanticViolation) Same(other SemanticViolation) bool {
	return v.Invariant != "" && v.Invariant == other.Invariant
}

// TrialKind classifies a candidate schedule relative to a target violation.
type TrialKind int

const (
	// TrialInvalid — schedule is not a legal execution of the scenario.
	TrialInvalid TrialKind = iota
	// TrialNoViolation — legal execution that does not fail.
	TrialNoViolation
	// TrialDifferentViolation — fails, but not the target semantic violation.
	TrialDifferentViolation
	// TrialSameViolation — fails with the same semantic violation (removable).
	TrialSameViolation
)

// Pair keeps the raw explore evidence and the derived counterexample.
// Original is never overwritten by Minimize.
type Pair struct {
	Original       witness.Witness
	Minimized      witness.Witness
	Removed        int
	ReplayAttempts int
}

// Minimize returns a 1-minimal witness that still violates the same semantic
// property as original. Original is retained verbatim in Pair.Original.
func Minimize(sc scenario.Scenario, specs []invariant.Spec, original witness.Witness) (Pair, error) {
	if len(original.Schedule) == 0 {
		return Pair{}, fmt.Errorf("minimize: empty schedule")
	}
	target := OfWitness(original)
	if target.Invariant == "" {
		return Pair{}, fmt.Errorf("minimize: missing invariant")
	}

	attempts := 0
	kind, verified, n := evaluateTrial(sc, specs, target, original.Schedule)
	attempts += n
	if kind != TrialSameViolation || verified == nil {
		return Pair{}, fmt.Errorf("minimize: original witness does not reproduce semantic violation %q", target.Invariant)
	}

	raw := copyWitness(original) // intact explore evidence
	sched := copySchedule(original.Schedule)
	current := *verified // canonical failedAt/schedule for the starting point
	current.Invariant = target.Invariant

	for {
		removedOne := false
		for i := 0; i < len(sched); i++ {
			cand := removeAt(sched, i)
			if len(cand) == 0 {
				continue
			}
			kind, got, n := evaluateTrial(sc, specs, target, cand)
			attempts += n
			if kind == TrialSameViolation && got != nil {
				sched = cand
				current = *got
				current.Invariant = target.Invariant
				removedOne = true
				break // restart from the beginning — deterministic 1-minimality
			}
			// invalid / no violation / different violation → transition required
		}
		if !removedOne {
			break
		}
	}

	return Pair{
		Original:       raw,
		Minimized:      current,
		Removed:        len(raw.Schedule) - len(current.Schedule),
		ReplayAttempts: attempts,
	}, nil
}

// SameViolation reports whether two witnesses share the same semantic failure.
func SameViolation(a, b witness.Witness) bool {
	return OfWitness(a).Same(OfWitness(b))
}

// evaluateTrial classifies a candidate schedule. attempts is always 1 when replay runs.
func evaluateTrial(sc scenario.Scenario, specs []invariant.Spec, target SemanticViolation, sched scenario.Schedule) (TrialKind, *witness.Witness, int) {
	res, err := replay.Run(sc, specs, witness.Witness{
		Schedule:  sched,
		Invariant: target.Invariant,
	})
	if err != nil {
		return TrialInvalid, nil, 1
	}
	if res.Passed || res.Witness == nil {
		return TrialNoViolation, nil, 1
	}
	if !OfWitness(*res.Witness).Same(target) {
		return TrialDifferentViolation, res.Witness, 1
	}
	return TrialSameViolation, res.Witness, 1
}

func removeAt(sched scenario.Schedule, i int) scenario.Schedule {
	out := make(scenario.Schedule, 0, len(sched)-1)
	out = append(out, sched[:i]...)
	out = append(out, sched[i+1:]...)
	return out
}

func copySchedule(s scenario.Schedule) scenario.Schedule {
	out := make(scenario.Schedule, len(s))
	copy(out, s)
	return out
}

func copyWitness(w witness.Witness) witness.Witness {
	return witness.Witness{
		Schedule:  copySchedule(w.Schedule),
		Invariant: w.Invariant,
		Violation: w.Violation,
		FailedAt:  w.FailedAt,
	}
}
