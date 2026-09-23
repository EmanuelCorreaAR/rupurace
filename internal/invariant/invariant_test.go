package invariant_test

import (
	"errors"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

func TestCheck_Pass(t *testing.T) {
	specs := []invariant.Spec{{
		Name: "always",
		Hold: func(scenario.State) error { return nil },
	}}
	if w := invariant.Check(nil, nil, specs); w != nil {
		t.Fatalf("expected nil witness, got %+v", w)
	}
}

func TestCheck_FailCapturesScheduleInvariantFailedAt(t *testing.T) {
	sched := scenario.Schedule{
		{Worker: "A", Step: 0},
		{Worker: "B", Step: 0},
		{Worker: "A", Step: 1},
	}
	specs := []invariant.Spec{{
		Name: "owners <= 1",
		Hold: func(scenario.State) error { return errors.New("owners <= 1") },
	}}

	w := invariant.Check("state", sched, specs)
	if w == nil {
		t.Fatal("expected witness")
	}
	if w.Invariant != "owners <= 1" {
		t.Fatalf("invariant: %q", w.Invariant)
	}
	if w.Violation != "owners <= 1" {
		t.Fatalf("violation: %q", w.Violation)
	}
	if w.FailedAt != 3 {
		t.Fatalf("failedAt: %d", w.FailedAt)
	}
	if len(w.Schedule) != 3 {
		t.Fatalf("schedule len: %d", len(w.Schedule))
	}
	// Witness must own a copy — mutating input must not alias.
	sched[0].Worker = "Z"
	if w.Schedule[0].Worker == "Z" {
		t.Fatal("witness schedule aliases input schedule")
	}
}

func TestCheck_UsesErrorAsNameWhenEmpty(t *testing.T) {
	specs := []invariant.Spec{{
		Hold: func(scenario.State) error { return errors.New("raw failure") },
	}}
	w := invariant.Check(nil, scenario.Schedule{{Worker: "A", Step: 0}}, specs)
	if w == nil || w.Invariant != "raw failure" {
		t.Fatalf("expected invariant from error, got %+v", w)
	}
}
