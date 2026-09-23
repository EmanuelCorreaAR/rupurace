package scheduler_test

import (
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

func TestExhaustive_DoubleWithdraw_FindsBug(t *testing.T) {
	loaded, err := demos.Load("double-withdraw", map[string]any{
		"balance": 100,
		"amount":  60,
	})
	if err != nil {
		t.Fatal(err)
	}

	found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	if found.Passed || found.Witness == nil {
		t.Fatalf("expected a violation, got %+v", found)
	}
	if found.Witness.Invariant != "balance >= 0" {
		t.Fatalf("invariant: %q", found.Witness.Invariant)
	}
	if found.Witness.FailedAt != len(found.Witness.Schedule) {
		t.Fatalf("failedAt=%d len=%d", found.Witness.FailedAt, len(found.Witness.Schedule))
	}
}

func TestExhaustive_DeterministicWitness(t *testing.T) {
	loaded, err := demos.Load("double-withdraw", nil)
	if err != nil {
		t.Fatal(err)
	}

	first := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	for i := 0; i < 20; i++ {
		got := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
		if !witnessesEqual(first.Witness, got.Witness) {
			t.Fatalf("run %d diverged\nfirst=%+v\ngot=%+v", i+1, first.Witness, got.Witness)
		}
	}
}

func TestReplay_SameFailureEveryTime(t *testing.T) {
	loaded, err := demos.Load("double-withdraw", nil)
	if err != nil {
		t.Fatal(err)
	}

	found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	if found.Witness == nil {
		t.Fatal("expected witness")
	}

	var first witness.Result
	for i := 0; i < 20; i++ {
		got, err := replay.Run(loaded.Scene, loaded.Specs, *found.Witness)
		if err != nil {
			t.Fatalf("replay: %v", err)
		}
		if got.Passed || got.Witness == nil {
			t.Fatalf("replay should fail, got %+v", got)
		}
		if got.Witness.FailedAt != found.Witness.FailedAt {
			t.Fatalf("failedAt mismatch: explore=%d replay=%d", found.Witness.FailedAt, got.Witness.FailedAt)
		}
		if got.Witness.Invariant != found.Witness.Invariant {
			t.Fatalf("invariant mismatch: %q vs %q", found.Witness.Invariant, got.Witness.Invariant)
		}
		if i == 0 {
			first = got
			continue
		}
		if !witnessesEqual(first.Witness, got.Witness) {
			t.Fatalf("replay %d diverged", i+1)
		}
	}
}

func TestSeeded_Deterministic(t *testing.T) {
	loaded, err := demos.Load("counter", map[string]any{
		"quota_a": 3, "quota_b": 3, "max_value": 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := scheduler.Config{Seed: 42, MaxSteps: 64}
	first := scheduler.Explore(loaded.Scene, loaded.Specs, cfg)
	for i := 0; i < 10; i++ {
		got := scheduler.Explore(loaded.Scene, loaded.Specs, cfg)
		if first.Passed != got.Passed || len(first.Schedule) != len(got.Schedule) {
			t.Fatalf("diverged on run %d", i+1)
		}
		for j := range first.Schedule {
			if first.Schedule[j] != got.Schedule[j] {
				t.Fatalf("schedule diverged at %d", j)
			}
		}
	}
}

func witnessesEqual(a, b *witness.Witness) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Invariant != b.Invariant || a.FailedAt != b.FailedAt || a.Violation != b.Violation {
		return false
	}
	if len(a.Schedule) != len(b.Schedule) {
		return false
	}
	for i := range a.Schedule {
		if a.Schedule[i] != b.Schedule[i] {
			return false
		}
	}
	return true
}
