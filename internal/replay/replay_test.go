package replay_test

import (
	"errors"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
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

func TestReplay_ReproducesWitness(t *testing.T) {
	sc := scenario.Counter{QuotaA: 3, QuotaB: 3}
	// Limit 2: any third increment violates.
	invs := []invariant.Invariant{maxValue(2)}
	cfg := scheduler.Config{Seed: 7, MaxSteps: 64}

	found := scheduler.Explore(sc, invs, cfg)
	if found.Passed || found.Witness == nil {
		t.Fatalf("expected a violation witness, got %+v", found)
	}

	replayed, err := replay.Run(sc, invs, *found.Witness)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replayed.Passed {
		t.Fatal("replay should reproduce the violation")
	}
	if !schedulesEqual(found.Witness.Schedule, replayed.Witness.Schedule) {
		t.Fatalf("replay schedule mismatch\nexplore=%v\nreplay=%v",
			found.Witness.Schedule, replayed.Witness.Schedule)
	}
	if found.Witness.Violation != replayed.Witness.Violation {
		t.Fatalf("violation mismatch: %q vs %q", found.Witness.Violation, replayed.Witness.Violation)
	}
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
