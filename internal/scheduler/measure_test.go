package scheduler_test

import (
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
)

func TestMultinomial_TwoByFive(t *testing.T) {
	got := scheduler.Multinomial([]int{5, 5})
	if got != 252 {
		t.Fatalf("2×5 interleavings: got %d want 252", got)
	}
}

func TestMultinomial_TwoByThree(t *testing.T) {
	got := scheduler.Multinomial([]int{3, 3})
	if got != 20 {
		t.Fatalf("2×3 interleavings: got %d want 20", got)
	}
}

func TestMeasure_InterleaveExploresFullSpace(t *testing.T) {
	loaded, err := demos.Load("interleave", map[string]any{"workers": 2, "steps": 5})
	if err != nil {
		t.Fatal(err)
	}
	cfg := scheduler.ExhaustiveConfig{
		Measure:                true,
		ContinueAfterViolation: true,
		Workers:                2,
		StepsPerWorker:         5,
	}
	res := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, cfg)
	if !res.Passed {
		t.Fatalf("interleave should not violate: %+v", res.Witness)
	}
	if res.Stats == nil {
		t.Fatal("expected stats")
	}
	if res.Stats.PossibleSchedules != 252 {
		t.Fatalf("possible: %d", res.Stats.PossibleSchedules)
	}
	if res.Stats.SchedulesExplored != 252 {
		t.Fatalf("explored: %d want 252", res.Stats.SchedulesExplored)
	}
	if res.Stats.UniqueStates < 1 || res.Stats.StatesVisited < res.Stats.UniqueStates {
		t.Fatalf("states visited=%d unique=%d", res.Stats.StatesVisited, res.Stats.UniqueStates)
	}
	if res.Stats.EquivalentPruned != 0 {
		t.Fatalf("pruning not implemented yet, got %d", res.Stats.EquivalentPruned)
	}
	if res.Stats.Violations != 0 {
		t.Fatalf("violations: %d", res.Stats.Violations)
	}
}

func TestMeasure_StatsDeterministic(t *testing.T) {
	loaded, err := demos.Load("interleave", map[string]any{"workers": 2, "steps": 4})
	if err != nil {
		t.Fatal(err)
	}
	cfg := scheduler.ExhaustiveConfig{
		Measure:        true,
		Workers:        2,
		StepsPerWorker: 4,
	}
	first := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, cfg)
	for i := 0; i < 10; i++ {
		got := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, cfg)
		if first.Stats.PossibleSchedules != got.Stats.PossibleSchedules ||
			first.Stats.SchedulesExplored != got.Stats.SchedulesExplored ||
			first.Stats.StatesVisited != got.Stats.StatesVisited ||
			first.Stats.UniqueStates != got.Stats.UniqueStates ||
			first.Stats.Violations != got.Stats.Violations {
			t.Fatalf("stats diverged on run %d\nfirst=%+v\ngot=%+v", i+1, first.Stats, got.Stats)
		}
	}
}

func TestMeasure_ContinueCountsViolations(t *testing.T) {
	loaded, err := demos.Load("check-then-act", nil)
	if err != nil {
		t.Fatal(err)
	}
	res := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{
		Measure:                true,
		ContinueAfterViolation: true,
	})
	if res.Stats == nil || res.Stats.Violations < 1 {
		t.Fatalf("expected ≥1 violation, stats=%+v", res.Stats)
	}
	if res.Witness == nil {
		t.Fatal("first witness still required")
	}
}

func TestConstitution_DefaultStillStopsOnFirstViolation(t *testing.T) {
	loaded, err := demos.Load("check-then-act", nil)
	if err != nil {
		t.Fatal(err)
	}
	res := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	if res.Passed || res.Witness == nil {
		t.Fatal("expected first violation")
	}
	// Without --continue, exhaustive returns early; Explored may be 0.
	if res.Stats != nil {
		t.Fatal("default path should not attach stats")
	}
}
