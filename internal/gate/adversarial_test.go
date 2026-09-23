package gate_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/EmanuelCorreaAR/rupurace/internal/gate"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

func TestAdversarial_EarlyReturn_NoWaiter(t *testing.T) {
	c := gate.NewController()
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "A", Step: 0})
		// early return — never reaches A:1
	})
	c.Quiesce()
	if err := c.Release(scenario.Transition{Worker: "A", Step: 0}); err != nil {
		t.Fatal(err)
	}
	c.Quiesce()
	err := c.Release(scenario.Transition{Worker: "A", Step: 1})
	if !errors.Is(err, gate.ErrNoWaiter) {
		t.Fatalf("want ErrNoWaiter, got %v", err)
	}
}

func TestAdversarial_GateNeverReached(t *testing.T) {
	c := gate.NewController()
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "A", Step: 0})
	})
	c.Quiesce()
	err := c.Release(scenario.Transition{Worker: "B", Step: 0})
	if !errors.Is(err, gate.ErrNoWaiter) {
		t.Fatalf("want ErrNoWaiter for never-reached gate, got %v (parked=%v)", err, c.ParkedKeys())
	}
}

func TestAdversarial_DoubleRelease(t *testing.T) {
	c := gate.NewController()
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "A", Step: 0})
	})
	c.Quiesce()
	t0 := scenario.Transition{Worker: "A", Step: 0}
	if err := c.Release(t0); err != nil {
		t.Fatal(err)
	}
	c.Quiesce()
	err := c.Release(t0)
	if !errors.Is(err, gate.ErrAlreadyReleased) {
		t.Fatalf("want ErrAlreadyReleased, got %v", err)
	}
}

func TestAdversarial_PanicRecovered(t *testing.T) {
	c := gate.NewController()
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "A", Step: 0})
		panic("boom")
	})
	c.Quiesce()
	if err := c.Release(scenario.Transition{Worker: "A", Step: 0}); err != nil {
		t.Fatal(err)
	}
	c.Quiesce()
	p := c.Panicked()
	if len(p) != 1 || p[0] != "boom" {
		t.Fatalf("panicked=%v", p)
	}
}

func TestAdversarial_LifecycleExitWithoutGates(t *testing.T) {
	c := gate.NewController()
	c.Go(func() {}) // exits immediately
	c.Quiesce()     // must settle on exit, not hang
	if keys := c.ParkedKeys(); len(keys) != 0 {
		t.Fatalf("parked=%v", keys)
	}
}

func TestAdversarial_NestedAwaits(t *testing.T) {
	c := gate.NewController()
	order := make([]string, 0, 3)
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "A", Step: 0})
		order = append(order, "a0")
		c.Await(scenario.Transition{Worker: "A", Step: 1})
		order = append(order, "a1")
		c.Await(scenario.Transition{Worker: "A", Step: 2})
		order = append(order, "a2")
	})
	c.Quiesce()
	for _, step := range []int{0, 1, 2} {
		if err := c.Release(scenario.Transition{Worker: "A", Step: step}); err != nil {
			t.Fatal(err)
		}
		c.Quiesce()
	}
	want := []string{"a0", "a1", "a2"}
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Fatalf("order=%v", order)
	}
}

func TestAdversarial_ChannelBetweenGates(t *testing.T) {
	// Channel rendezvous is fine if it completes between Awaits (or both sides
	// are released so the send/recv can finish before the next Quiesce).
	c := gate.NewController()
	ch := make(chan int, 1)
	var got int
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "A", Step: 0})
		ch <- 7
		c.Await(scenario.Transition{Worker: "A", Step: 1})
	})
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "B", Step: 0})
		got = <-ch
		c.Await(scenario.Transition{Worker: "B", Step: 1})
	})
	c.Quiesce()
	// Open both step-0 gates so the buffered/sync handshake can complete.
	if err := c.Release(scenario.Transition{Worker: "A", Step: 0}); err != nil {
		t.Fatal(err)
	}
	if err := c.Release(scenario.Transition{Worker: "B", Step: 0}); err != nil {
		t.Fatal(err)
	}
	c.Quiesce()
	if got != 7 {
		t.Fatalf("got=%d", got)
	}
	if err := c.Release(scenario.Transition{Worker: "A", Step: 1}); err != nil {
		t.Fatal(err)
	}
	if err := c.Release(scenario.Transition{Worker: "B", Step: 1}); err != nil {
		t.Fatal(err)
	}
	c.Quiesce()
}

func TestAdversarial_BlockingChannelIsNotQuiescence(t *testing.T) {
	// Documented limitation: blocked on chan recv (not Await) ⇒ Quiesce hangs.
	c := gate.NewController()
	ch := make(chan struct{})
	c.Go(func() {
		<-ch // not an Await
	})
	done := make(chan struct{})
	go func() {
		c.Quiesce()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("Quiesce returned while worker blocked on channel — would hide deadlocks")
	case <-time.After(50 * time.Millisecond):
		// expected: not quiescence
	}
	close(ch) // unblock so the test can exit
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Quiesce did not return after channel close")
	}
}

func TestAdversarial_ContextCancelDoesNotUnblockAwait(t *testing.T) {
	// Await is not context-aware (0.4.1): cancel alone must not release a gate.
	ctx, cancel := context.WithCancel(context.Background())
	c := gate.NewController()
	c.Go(func() {
		c.Await(scenario.Transition{Worker: "A", Step: 0})
		_ = ctx
	})
	c.Quiesce()
	cancel()
	time.Sleep(20 * time.Millisecond)
	if keys := c.ParkedKeys(); len(keys) != 1 || keys[0] != "A:0" {
		t.Fatalf("cancel must not clear park: %v", keys)
	}
	if err := c.Release(scenario.Transition{Worker: "A", Step: 0}); err != nil {
		t.Fatal(err)
	}
	c.Quiesce()
}

func TestAdversarial_ReplayEarlyExitSurfacesError(t *testing.T) {
	p := gate.Program{
		Workers: []scenario.WorkerID{"A"},
		Steps:   2,
		Start: func(c *gate.Controller) {
			c.Go(func() {
				c.Await(scenario.Transition{Worker: "A", Step: 0})
			})
		},
		Observe: func(c *gate.Controller) scenario.State { return nil },
	}
	sched := scenario.Schedule{
		{Worker: "A", Step: 0},
		{Worker: "A", Step: 1},
	}
	_, err := gate.Replay(p, nil, sched)
	if !errors.Is(err, gate.ErrNoWaiter) {
		t.Fatalf("want ErrNoWaiter wrapped, got %v", err)
	}
}

func TestAdversarial_MutexBlockIsNotQuiescence(t *testing.T) {
	c := gate.NewController()
	var mu sync.Mutex
	mu.Lock()
	c.Go(func() {
		mu.Lock() // never durable / never Await
		mu.Unlock()
	})
	done := make(chan struct{})
	go func() {
		c.Quiesce()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("Quiesce returned while worker blocked on Mutex")
	case <-time.After(50 * time.Millisecond):
	}
	mu.Unlock()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Quiesce did not return after Unlock")
	}
}
