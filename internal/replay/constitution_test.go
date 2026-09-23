package replay_test

import (
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
)

// TestExploreWitnessReplaysSameFailure is the constitutional property of RupuRace:
//
//	Explore(S) → Witness W
//	Replay(S, W.schedule) → Failure F
//
//	F.Invariant == W.Invariant
//	F.FailedAt  == W.FailedAt
//	F.Schedule  == W.Schedule
func TestExploreWitnessReplaysSameFailure(t *testing.T) {
	for _, name := range []string{
		"double-withdraw",
		"lost-update",
		"check-then-act",
		"init-ordering",
	} {
		t.Run(name, func(t *testing.T) {
			loaded, err := demos.Load(name, nil)
			if err != nil {
				t.Fatal(err)
			}

			explored := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
			if explored.Passed || explored.Witness == nil {
				t.Fatalf("explore must find a violation for %s", name)
			}
			w := explored.Witness

			failed, err := replay.Run(loaded.Scene, loaded.Specs, *w)
			if err != nil {
				t.Fatalf("replay: %v", err)
			}
			if failed.Passed || failed.Witness == nil {
				t.Fatalf("replay must reproduce the failure: %+v", failed)
			}
			f := failed.Witness

			if f.Invariant != w.Invariant {
				t.Fatalf("invariant: explore=%q replay=%q", w.Invariant, f.Invariant)
			}
			if f.FailedAt != w.FailedAt {
				t.Fatalf("failedAt: explore=%d replay=%d", w.FailedAt, f.FailedAt)
			}
			if len(f.Schedule) != len(w.Schedule) {
				t.Fatalf("schedule len: explore=%d replay=%d", len(w.Schedule), len(f.Schedule))
			}
			for i := range w.Schedule {
				if f.Schedule[i] != w.Schedule[i] {
					t.Fatalf("schedule[%d]: explore=%s replay=%s", i, w.Schedule[i], f.Schedule[i])
				}
			}
		})
	}
}

func TestReplay_RejectsDisabledTransition(t *testing.T) {
	loaded, err := demos.Load("check-then-act", nil)
	if err != nil {
		t.Fatal(err)
	}
	explored := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	if explored.Witness == nil {
		t.Fatal("expected witness")
	}
	w := *explored.Witness
	// Corrupt: jump to B:1 before B:0.
	if len(w.Schedule) < 2 {
		t.Fatal("need a multi-step schedule")
	}
	w.Schedule[0].Worker = "B"
	w.Schedule[0].Step = 1

	_, err = replay.Run(loaded.Scene, loaded.Specs, w)
	if err == nil {
		t.Fatal("expected error for disabled transition")
	}
}
