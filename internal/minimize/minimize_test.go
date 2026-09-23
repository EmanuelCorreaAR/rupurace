package minimize_test

import (
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/minimize"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// TestMinimize_Constitution is the 0.3.0 thesis:
//
//	W = Explore(S)
//	M = Minimize(S, W)
//	Replay(W) and Replay(M) fail
//	SameViolation(W, M)
//	len(M) <= len(W)
//	∀ t ∈ M: remove(t) ⇒ invalid ∨ !SameViolation
func TestMinimize_Constitution(t *testing.T) {
	for _, name := range []string{"double-withdraw", "lost-update", "check-then-act", "init-ordering"} {
		t.Run(name, func(t *testing.T) {
			loaded, err := demos.Load(name, nil)
			if err != nil {
				t.Fatal(err)
			}
			found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
			if found.Witness == nil {
				t.Fatal("expected witness")
			}
			w := *found.Witness

			pair, err := minimize.Minimize(loaded.Scene, loaded.Specs, w)
			if err != nil {
				t.Fatal(err)
			}
			m := pair.Minimized

			// Original evidence retained (explore schedule intact).
			if !schedulesEqual(pair.Original.Schedule, w.Schedule) || pair.Original.Invariant != w.Invariant {
				t.Fatal("original witness was mutated")
			}

			rw, err := replay.Run(loaded.Scene, loaded.Specs, pair.Original)
			if err != nil || rw.Passed || rw.Witness == nil {
				t.Fatalf("Replay(original) must fail: err=%v", err)
			}
			rm, err := replay.Run(loaded.Scene, loaded.Specs, m)
			if err != nil || rm.Passed || rm.Witness == nil {
				t.Fatalf("Replay(minimal) must fail: err=%v", err)
			}

			if !minimize.SameViolation(pair.Original, m) {
				t.Fatalf("SameViolation failed: %q vs %q", pair.Original.Invariant, m.Invariant)
			}
			if len(m.Schedule) > len(pair.Original.Schedule) {
				t.Fatalf("minimized longer than original")
			}

			// Replay(M) matches M's own failedAt/schedule (not original's).
			if rm.Witness.FailedAt != m.FailedAt || !schedulesEqual(rm.Witness.Schedule, m.Schedule) {
				t.Fatalf("Replay(M) must match M itself")
			}

			// 1-minimality
			for i := 0; i < len(m.Schedule); i++ {
				trial := removeAt(m.Schedule, i)
				if len(trial) == 0 {
					continue
				}
				res, err := replay.Run(loaded.Scene, loaded.Specs, witness.Witness{
					Schedule:  trial,
					Invariant: m.Invariant,
				})
				if err != nil {
					continue // invalid → transition required
				}
				if res.Passed || res.Witness == nil {
					continue // no violation → required
				}
				if minimize.SameViolation(m, *res.Witness) {
					t.Fatalf("not 1-minimal: removed index %d still SameViolation\ntrial=%v", i, trial)
				}
			}
		})
	}
}

func TestMinimize_Deterministic(t *testing.T) {
	loaded, err := demos.Load("check-then-act", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	first, err := minimize.Minimize(loaded.Scene, loaded.Specs, *found.Witness)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		got, err := minimize.Minimize(loaded.Scene, loaded.Specs, *found.Witness)
		if err != nil {
			t.Fatal(err)
		}
		if !schedulesEqual(first.Minimized.Schedule, got.Minimized.Schedule) ||
			first.Minimized.FailedAt != got.Minimized.FailedAt ||
			first.Minimized.Invariant != got.Minimized.Invariant ||
			first.Removed != got.Removed {
			t.Fatalf("Minimize diverged on run %d", i+1)
		}
	}
}

func TestMinimize_RemovesIrrelevantSuffix(t *testing.T) {
	// Replay stops at the first violation, so a padded suffix is noise —
	// Minimize must strip it while keeping SameViolation.
	loaded, err := demos.Load("check-then-act", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	padded := *found.Witness
	padded.Schedule = append(append(scenario.Schedule{}, padded.Schedule...),
		scenario.Transition{Worker: "noise", Step: 0},
		scenario.Transition{Worker: "noise", Step: 1},
	)

	pair, err := minimize.Minimize(loaded.Scene, loaded.Specs, padded)
	if err != nil {
		t.Fatal(err)
	}
	if pair.Removed < 2 {
		t.Fatalf("expected suffix stripped, removed=%d schedule=%v", pair.Removed, pair.Minimized.Schedule)
	}
	if len(pair.Minimized.Schedule) > len(found.Witness.Schedule) {
		t.Fatalf("minimized longer than real witness")
	}
	if !minimize.SameViolation(padded, pair.Minimized) {
		t.Fatal("SameViolation")
	}
	if len(pair.Original.Schedule) != len(padded.Schedule) {
		t.Fatal("original must stay intact including padding")
	}
}

func TestMinimize_FailedAtMayDifferFromOriginal(t *testing.T) {
	loaded, err := demos.Load("lost-update", nil)
	if err != nil {
		t.Fatal(err)
	}
	found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	pair, err := minimize.Minimize(loaded.Scene, loaded.Specs, *found.Witness)
	if err != nil {
		t.Fatal(err)
	}
	if pair.Minimized.FailedAt != len(pair.Minimized.Schedule) {
		t.Fatalf("minimized failedAt should equal its schedule length")
	}
	// Semantic identity is invariant, not failedAt equality with original.
	if !minimize.SameViolation(pair.Original, pair.Minimized) {
		t.Fatal("expected SameViolation")
	}
}

func TestSemanticViolation_IgnoresFailedAt(t *testing.T) {
	a := witness.Witness{Invariant: "owners <= 1", FailedAt: 6}
	b := witness.Witness{Invariant: "owners <= 1", FailedAt: 4}
	if !minimize.SameViolation(a, b) {
		t.Fatal("SameViolation must ignore failedAt")
	}
	c := witness.Witness{Invariant: "balance >= 0", FailedAt: 4}
	if minimize.SameViolation(a, c) {
		t.Fatal("different invariants must not match")
	}
}

func removeAt(sched scenario.Schedule, i int) scenario.Schedule {
	out := make(scenario.Schedule, 0, len(sched)-1)
	out = append(out, sched[:i]...)
	out = append(out, sched[i+1:]...)
	return out
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
