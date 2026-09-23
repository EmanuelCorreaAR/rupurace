package gate_test

import (
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/gate"
	"github.com/EmanuelCorreaAR/rupurace/internal/minimize"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// 0.4 thesis: real goroutines + Await gates enter the consolidated pipeline.
//
//	Explore → W → Replay(W) → Minimize → Replay(M)
func TestGate_CheckThenAct_Pipeline(t *testing.T) {
	p := gate.CheckThenAct()
	specs := gate.OwnersSpec()

	found, err := gate.Explore(p, specs)
	if err != nil {
		t.Fatal(err)
	}
	if found.Passed || found.Witness == nil {
		t.Fatal("expected violation from live check-then-act")
	}
	w := *found.Witness
	if w.Invariant != "owners <= 1" {
		t.Fatalf("invariant: %q", w.Invariant)
	}
	t.Logf("original schedule=%v failedAt=%d", w.Schedule, w.FailedAt)

	// Replay via gate.Replay
	rw, err := gate.Replay(p, specs, w.Schedule)
	if err != nil || rw.Passed || rw.Witness == nil {
		t.Fatalf("gate.Replay: err=%v passed=%v", err, rw.Passed)
	}
	if !minimize.SameViolation(w, *rw.Witness) {
		t.Fatal("Replay SameViolation")
	}

	// Replay via scenario adapter (witness_v1 path)
	sc := gate.ReplayScenario(p)
	rr, err := replay.Run(sc, specs, w)
	if err != nil || rr.Passed || rr.Witness == nil {
		t.Fatalf("replay.Run adapter: err=%v passed=%v", err, rr.Passed)
	}
	if !minimize.SameViolation(w, *rr.Witness) {
		t.Fatal("adapter SameViolation")
	}

	// Minimize through the adapter — same witness_v1 machinery
	pair, err := minimize.Minimize(sc, specs, w)
	if err != nil {
		t.Fatal(err)
	}
	if !minimize.SameViolation(pair.Original, pair.Minimized) {
		t.Fatal("minimize SameViolation")
	}
	if len(pair.Minimized.Schedule) > len(pair.Original.Schedule) {
		t.Fatal("minimized longer")
	}

	rm, err := gate.Replay(p, specs, pair.Minimized.Schedule)
	if err != nil || rm.Passed {
		t.Fatalf("Replay(minimal): err=%v passed=%v", err, rm.Passed)
	}

	// 1-minimality
	m := pair.Minimized
	for i := range m.Schedule {
		trial := append(scenario.Schedule{}, m.Schedule[:i]...)
		trial = append(trial, m.Schedule[i+1:]...)
		if len(trial) == 0 {
			continue
		}
		res, err := replay.Run(gate.ReplayScenario(p), specs, witness.Witness{
			Schedule:  trial,
			Invariant: m.Invariant,
		})
		if err != nil {
			continue
		}
		if !res.Passed && res.Witness != nil && minimize.SameViolation(m, *res.Witness) {
			t.Fatalf("not 1-minimal at %d trial=%v", i, trial)
		}
	}
}

func TestGate_DeterministicExplore(t *testing.T) {
	p := gate.CheckThenAct()
	specs := gate.OwnersSpec()
	first, err := gate.Explore(p, specs)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		got, err := gate.Explore(p, specs)
		if err != nil {
			t.Fatal(err)
		}
		if !schedulesEqual(first.Witness.Schedule, got.Witness.Schedule) ||
			first.Witness.FailedAt != got.Witness.FailedAt {
			t.Fatalf("Explore diverged on run %d: %v vs %v", i+1, first.Witness.Schedule, got.Witness.Schedule)
		}
	}
}

func TestGate_SafeSchedulePasses(t *testing.T) {
	p := gate.CheckThenAct()
	specs := gate.OwnersSpec()
	safe := scenario.Schedule{
		{Worker: "A", Step: 0},
		{Worker: "A", Step: 1},
		{Worker: "B", Step: 0},
		{Worker: "B", Step: 1},
	}
	res, err := gate.Replay(p, specs, safe)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Passed {
		t.Fatalf("safe schedule should pass, got %+v", res.Witness)
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
