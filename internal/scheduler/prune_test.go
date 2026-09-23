package scheduler_test

import (
	"strings"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
)

func TestPrune_InterleaveReducesWork(t *testing.T) {
	loaded, err := demos.Load("interleave", map[string]any{"workers": 2, "steps": 5})
	if err != nil {
		t.Fatal(err)
	}

	base := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{
		Measure:        true,
		Workers:        2,
		StepsPerWorker: 5,
	})
	pruned := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{
		Measure:        true,
		Prune:          true,
		Workers:        2,
		StepsPerWorker: 5,
	})

	if base.Stats == nil || pruned.Stats == nil {
		t.Fatal("expected stats")
	}
	if base.Stats.EquivalentPruned != 0 {
		t.Fatalf("baseline pruned=%d", base.Stats.EquivalentPruned)
	}
	if pruned.Stats.EquivalentPruned <= 0 {
		t.Fatalf("expected EquivalentPruned > 0, got %d", pruned.Stats.EquivalentPruned)
	}
	if pruned.Stats.SchedulesExplored >= base.Stats.SchedulesExplored {
		t.Fatalf("schedules executed did not drop: base=%d pruned=%d",
			base.Stats.SchedulesExplored, pruned.Stats.SchedulesExplored)
	}
	if pruned.Stats.StatesVisited >= base.Stats.StatesVisited {
		t.Fatalf("states visited did not drop: base=%d pruned=%d",
			base.Stats.StatesVisited, pruned.Stats.StatesVisited)
	}
	if pruned.Stats.PossibleSchedules != 252 || base.Stats.PossibleSchedules != 252 {
		t.Fatalf("possible schedules changed")
	}
	if pruned.Stats.UniqueStates != base.Stats.UniqueStates {
		t.Fatalf("unique states should match lattice size: base=%d pruned=%d",
			base.Stats.UniqueStates, pruned.Stats.UniqueStates)
	}
}

func TestPrune_SameFirstWitness(t *testing.T) {
	for _, name := range []string{"double-withdraw", "lost-update", "check-then-act", "init-ordering"} {
		t.Run(name, func(t *testing.T) {
			loaded, err := demos.Load(name, nil)
			if err != nil {
				t.Fatal(err)
			}
			plain := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
			withPrune := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{Prune: true})
			if plain.Witness == nil || withPrune.Witness == nil {
				t.Fatal("expected witnesses")
			}
			if plain.Witness.Invariant != withPrune.Witness.Invariant ||
				plain.Witness.FailedAt != withPrune.Witness.FailedAt ||
				!schedulesEqual(plain.Witness.Schedule, withPrune.Witness.Schedule) {
				t.Fatalf("first witness diverged\nplain=%+v\nprune=%+v", plain.Witness, withPrune.Witness)
			}

			got, err := replay.Run(loaded.Scene, loaded.Specs, *withPrune.Witness)
			if err != nil || got.Passed || got.Witness == nil {
				t.Fatalf("replay failed: err=%v result=%+v", err, got)
			}
			if got.Witness.FailedAt != withPrune.Witness.FailedAt {
				t.Fatalf("replay failedAt mismatch")
			}
		})
	}
}

func TestPrune_SameViolationCount(t *testing.T) {
	loaded, err := demos.Load("check-then-act", nil)
	if err != nil {
		t.Fatal(err)
	}
	plain := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{
		Measure:                true,
		ContinueAfterViolation: true,
	})
	withPrune := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{
		Measure:                true,
		ContinueAfterViolation: true,
		Prune:                  true,
	})
	// Pruning collapses commuting prefixes to the same node (A0 B0 ≡ B0 A0),
	// so path-hit Violations may drop — but the bug remains reachable.
	if plain.Stats.Violations < 1 || withPrune.Stats.Violations < 1 {
		t.Fatalf("both must find violations: plain=%d prune=%d",
			plain.Stats.Violations, withPrune.Stats.Violations)
	}
	if withPrune.Stats.Violations > plain.Stats.Violations {
		t.Fatalf("prune should not invent violations: plain=%d prune=%d",
			plain.Stats.Violations, withPrune.Stats.Violations)
	}
	if withPrune.Stats.EquivalentPruned <= 0 {
		t.Fatalf("expected pruning on check-then-act, got %d", withPrune.Stats.EquivalentPruned)
	}
	plainSet := collectViolationSchedules(t, false)
	pruneSet := collectViolationSchedules(t, true)
	for sched := range pruneSet {
		if _, ok := plainSet[sched]; !ok {
			t.Fatalf("prune found schedule absent from full search: %s", sched)
		}
	}
	if len(pruneSet) == 0 {
		t.Fatal("prune must keep at least one violating schedule")
	}
}

func collectViolationSchedules(t *testing.T, prune bool) map[string]struct{} {
	t.Helper()
	loaded, err := demos.Load("check-then-act", nil)
	if err != nil {
		t.Fatal(err)
	}
	type frame struct {
		state scenario.State
		sched scenario.Schedule
	}
	sc, specs := loaded.Scene, loaded.Specs
	stack := []frame{{state: sc.Initial()}}
	expanded := map[string]struct{}{}
	out := map[string]struct{}{}

	sortEn := func(en []scenario.Transition) []scenario.Transition {
		out := append([]scenario.Transition(nil), en...)
		for i := 0; i < len(out); i++ {
			for j := i + 1; j < len(out); j++ {
				if string(out[j].Worker) < string(out[i].Worker) ||
					(out[j].Worker == out[i].Worker && out[j].Step < out[i].Step) {
					out[i], out[j] = out[j], out[i]
				}
			}
		}
		return out
	}

	for len(stack) > 0 {
		fr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		en := sortEn(sc.Enabled(fr.state))
		if len(en) == 0 {
			continue
		}
		if prune {
			key := scheduler.NodeKey(fr.state, en)
			if _, ok := expanded[key]; ok {
				continue
			}
			expanded[key] = struct{}{}
		}
		for i := len(en) - 1; i >= 0; i-- {
			tr := en[i]
			ns := sc.Apply(fr.state, tr)
			nsch := append(fr.sched[:len(fr.sched):len(fr.sched)], tr)
			if w := invariant.Check(ns, nsch, specs); w != nil {
				var b strings.Builder
				for _, x := range nsch {
					b.WriteString(x.String())
					b.WriteByte(' ')
				}
				out[b.String()] = struct{}{}
				_ = w
				continue
			}
			stack = append(stack, frame{ns, nsch})
		}
	}
	return out
}

func TestNodeKey_DistinguishesProgress(t *testing.T) {
	// Same balance-like data is not enough: different enabled futures must differ.
	stA := scenario.DoubleWithdrawState{Balance: 100, NextA: 2, NextB: 1}
	stB := scenario.DoubleWithdrawState{Balance: 100, NextA: 1, NextB: 2}
	enA := []scenario.Transition{{Worker: "B", Step: 1}}
	enB := []scenario.Transition{{Worker: "A", Step: 1}}
	if scheduler.NodeKey(stA, enA) == scheduler.NodeKey(stB, enB) {
		t.Fatal("nodes with different progress must not share a prune key")
	}
}
