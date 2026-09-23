package scheduler_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// failingScenarios are cooperative demos that must expose at least one violation.
var failingScenarios = []string{
	"double-withdraw",
	"lost-update",
	"check-then-act",
	"init-ordering",
}

func TestHardening_ExhaustiveDeterministicAcrossScenarios(t *testing.T) {
	for _, name := range failingScenarios {
		t.Run(name, func(t *testing.T) {
			loaded := mustLoad(t, name)
			first := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
			if first.Passed || first.Witness == nil {
				t.Fatalf("expected violation for %s", name)
			}
			for i := 0; i < 50; i++ {
				got := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
				if !resultsEqual(first, got) {
					t.Fatalf("run %d diverged\nfirst=%+v\ngot=%+v", i+1, first.Witness, got.Witness)
				}
			}
		})
	}
}

func TestHardening_ReplayIdenticalFailedAt(t *testing.T) {
	for _, name := range failingScenarios {
		t.Run(name, func(t *testing.T) {
			loaded := mustLoad(t, name)
			found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
			if found.Witness == nil {
				t.Fatal("expected witness")
			}
			var baseline witness.Result
			for i := 0; i < 50; i++ {
				got, err := replay.Run(loaded.Scene, loaded.Specs, *found.Witness)
				if err != nil {
					t.Fatalf("replay: %v", err)
				}
				if got.Passed || got.Witness == nil {
					t.Fatalf("replay should fail: %+v", got)
				}
				if got.Witness.FailedAt != found.Witness.FailedAt {
					t.Fatalf("failedAt explore=%d replay=%d", found.Witness.FailedAt, got.Witness.FailedAt)
				}
				if got.Witness.Invariant != found.Witness.Invariant {
					t.Fatalf("invariant %q vs %q", found.Witness.Invariant, got.Witness.Invariant)
				}
				if !schedulesEqual(got.Witness.Schedule, found.Witness.Schedule) {
					t.Fatalf("schedule mismatch")
				}
				if i == 0 {
					baseline = got
					continue
				}
				if !resultsEqual(baseline, got) {
					t.Fatalf("replay %d diverged from baseline", i+1)
				}
			}
		})
	}
}

func TestHardening_ScrambledEnabledStillDeterministic(t *testing.T) {
	loaded := mustLoad(t, "check-then-act")
	scrambled := scrambleEnabled{inner: loaded.Scene}

	first := scheduler.ExploreExhaustive(scrambled, loaded.Specs, scheduler.ExhaustiveConfig{})
	plain := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	if !resultsEqual(first, plain) {
		t.Fatalf("scrambled Enabled changed witness\nscrambled=%+v\nplain=%+v", first.Witness, plain.Witness)
	}
	for i := 0; i < 20; i++ {
		got := scheduler.ExploreExhaustive(scrambled, loaded.Specs, scheduler.ExhaustiveConfig{})
		if !resultsEqual(first, got) {
			t.Fatalf("scrambled run %d diverged", i+1)
		}
	}
}

func TestHardening_ApplyDoesNotMutatePriorState(t *testing.T) {
	cases := []scenario.Scenario{
		scenario.DoubleWithdraw{Balance: 100, Amount: 60},
		scenario.LostUpdate{},
		scenario.CheckThenAct{},
		scenario.InitOrdering{Expected: 7},
	}
	for _, sc := range cases {
		t.Run(fmt.Sprintf("%T", sc), func(t *testing.T) {
			st := sc.Initial()
			enabled := sc.Enabled(st)
			if len(enabled) == 0 {
				t.Fatal("expected enabled transitions")
			}
			before := fmt.Sprintf("%#v", st)
			_ = sc.Apply(st, enabled[0])
			after := fmt.Sprintf("%#v", st)
			if before != after {
				t.Fatalf("Apply mutated prior state\nbefore=%s\nafter=%s", before, after)
			}
		})
	}
}

func TestHardening_SeededStable(t *testing.T) {
	loaded := mustLoad(t, "counter")
	cfg := scheduler.Config{Seed: 99, MaxSteps: 64}
	first := scheduler.Explore(loaded.Scene, loaded.Specs, cfg)
	for i := 0; i < 30; i++ {
		got := scheduler.Explore(loaded.Scene, loaded.Specs, cfg)
		if !resultsEqual(first, got) {
			t.Fatalf("seeded run %d diverged", i+1)
		}
	}
}

func TestHardening_AllNamedDemosLoad(t *testing.T) {
	for _, name := range demos.Names {
		if _, err := demos.Load(name, nil); err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
	}
}

func mustLoad(t *testing.T, name string) demos.Loaded {
	t.Helper()
	loaded, err := demos.Load(name, nil)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func resultsEqual(a, b witness.Result) bool {
	if a.Passed != b.Passed || a.Steps != b.Steps || a.Strategy != b.Strategy {
		return false
	}
	if !schedulesEqual(a.Schedule, b.Schedule) {
		return false
	}
	return witnessesEqual(a.Witness, b.Witness)
}

func witnessesEqual(a, b *witness.Witness) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Invariant != b.Invariant || a.FailedAt != b.FailedAt || a.Violation != b.Violation {
		return false
	}
	return schedulesEqual(a.Schedule, b.Schedule)
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

// scrambleEnabled reverses Enabled order to prove the scheduler normalizes it.
type scrambleEnabled struct {
	inner scenario.Scenario
}

func (s scrambleEnabled) Initial() scenario.State { return s.inner.Initial() }

func (s scrambleEnabled) Enabled(st scenario.State) []scenario.Transition {
	out := slices.Clone(s.inner.Enabled(st))
	slices.Reverse(out)
	return out
}

func (s scrambleEnabled) Apply(st scenario.State, t scenario.Transition) scenario.State {
	return s.inner.Apply(st, t)
}
