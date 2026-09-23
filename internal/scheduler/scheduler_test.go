package scheduler_test

import (
	"errors"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

func maxValue(limit int) invariant.Invariant {
	return func(st scenario.State) error {
		cs := st.(scenario.CounterState)
		if cs.Value > limit {
			return errors.New("counter exceeded limit")
		}
		return nil
	}
}

func TestExplore_Deterministic(t *testing.T) {
	sc := scenario.Counter{QuotaA: 3, QuotaB: 3}
	cfg := scheduler.Config{Seed: 42, MaxSteps: 64}
	invs := []invariant.Invariant{maxValue(100)}

	first := scheduler.Explore(sc, invs, cfg)
	for i := 0; i < 20; i++ {
		got := scheduler.Explore(sc, invs, cfg)
		if !resultsEqual(first, got) {
			t.Fatalf("run %d diverged from first result\nfirst=%+v\ngot=%+v", i+1, first, got)
		}
	}
}

func TestExplore_DifferentSeedsCanDiffer(t *testing.T) {
	sc := scenario.Counter{QuotaA: 4, QuotaB: 4}
	invs := []invariant.Invariant{maxValue(100)}

	a := scheduler.Explore(sc, invs, scheduler.Config{Seed: 1, MaxSteps: 64})
	b := scheduler.Explore(sc, invs, scheduler.Config{Seed: 2, MaxSteps: 64})

	if schedulesEqual(a.Schedule, b.Schedule) {
		t.Fatal("expected different seeds to produce different schedules for this scenario")
	}
}

func resultsEqual(a, b witness.Result) bool {
	if a.Passed != b.Passed || a.Steps != b.Steps {
		return false
	}
	if !schedulesEqual(a.Schedule, b.Schedule) {
		return false
	}
	if (a.Witness == nil) != (b.Witness == nil) {
		return false
	}
	if a.Witness == nil {
		return true
	}
	return a.Witness.Violation == b.Witness.Violation &&
		schedulesEqual(a.Witness.Schedule, b.Witness.Schedule)
}

func schedulesEqual(a, b scenario.Schedule) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
