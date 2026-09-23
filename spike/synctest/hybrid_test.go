package synctestspike_test

import (
	"testing"
	"testing/synctest"
)

// Hybrid sketch: real Go goroutines + explicit schedule gates.
// The "Explore" side picks an order of gate opens; synctest.Wait settles
// after each step. This is NOT free observation of Go's scheduler —
// it reintroduces cooperative control points, which is exactly RupuRace's model.
func TestSpike_HybridGatesReplayCheckThenAct(t *testing.T) {
	// Schedule that violates owners <= 1: A check, B check, A claim, B claim.
	sched := []string{"A:0", "B:0", "A:1", "B:1"}

	synctest.Test(t, func(t *testing.T) {
		type worker struct {
			check chan struct{}
			claim chan struct{}
			done  chan struct{}
		}
		newW := func() *worker {
			return &worker{
				check: make(chan struct{}),
				claim: make(chan struct{}),
				done:  make(chan struct{}),
			}
		}
		a, b := newW(), newW()
		owners := 0
		available := true

		run := func(w *worker) {
			defer close(w.done)
			<-w.check
			ok := available
			<-w.claim
			if ok {
				available = false
				owners++
			}
		}
		go run(a)
		go run(b)

		open := func(step string) {
			switch step {
			case "A:0":
				close(a.check)
			case "B:0":
				close(b.check)
			case "A:1":
				close(a.claim)
			case "B:1":
				close(b.claim)
			default:
				t.Fatalf("unknown step %s", step)
			}
			synctest.Wait()
		}

		for _, step := range sched {
			open(step)
		}
		<-a.done
		<-b.done

		if owners <= 1 {
			t.Fatalf("expected owners>1 for this schedule, got %d", owners)
		}
	})
}

// Same hybrid machinery with a safe schedule — proves we can steer outcomes.
func TestSpike_HybridGatesSafeSchedule(t *testing.T) {
	sched := []string{"A:0", "A:1", "B:0", "B:1"}

	synctest.Test(t, func(t *testing.T) {
		type worker struct {
			check, claim, done chan struct{}
		}
		newW := func() *worker {
			return &worker{
				check: make(chan struct{}),
				claim: make(chan struct{}),
				done:  make(chan struct{}),
			}
		}
		a, b := newW(), newW()
		owners := 0
		available := true

		run := func(w *worker) {
			defer close(w.done)
			<-w.check
			ok := available
			<-w.claim
			if ok {
				available = false
				owners++
			}
		}
		go run(a)
		go run(b)

		open := func(step string) {
			switch step {
			case "A:0":
				close(a.check)
			case "B:0":
				close(b.check)
			case "A:1":
				close(a.claim)
			case "B:1":
				close(b.claim)
			}
			synctest.Wait()
		}
		for _, step := range sched {
			open(step)
		}
		<-a.done
		<-b.done
		if owners != 1 {
			t.Fatalf("safe schedule: want owners=1, got %d", owners)
		}
	})
}
