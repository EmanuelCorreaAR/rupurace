// Package gate is the 0.4.x bridge from real Go goroutines into RupuRace's
// cooperative schedule model.
//
// A Gate is an observable transition: a goroutine parks until RupuRace releases
// that transition as part of a schedule. Naming is provisional (Gate / Await);
// the semantic is "this goroutine reached a controllable schedule point".
//
// synctest is not required here — Quiesce settles on park/exit. Use synctest in
// tests when you need a fake clock.
//
// Explore creates a fresh world per candidate schedule (side effects do not
// fork like immutable Scenario.Apply). ReplayScenario is for linear
// Replay/Minimize only — do not pass it to ExploreExhaustive.
package gate

import (
	"fmt"
	"sort"
	"sync"

	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

// Controller mediates named transitions between the SUT and the explorer.
type Controller struct {
	mu      sync.Mutex
	release map[string]chan struct{}
	parked  map[string]int
	living  int
	exited  int
	waiters []chan struct{}
	stash   any // per-world demo state (internal)
}

// NewController returns a fresh world controller.
func NewController() *Controller {
	return &Controller{
		release: make(map[string]chan struct{}),
		parked:  make(map[string]int),
	}
}

// Go starts a goroutine tracked for Quiesce.
func (c *Controller) Go(fn func()) {
	c.mu.Lock()
	c.living++
	c.mu.Unlock()
	go func() {
		defer func() {
			c.mu.Lock()
			c.exited++
			c.broadcastLocked()
			c.mu.Unlock()
		}()
		fn()
	}()
}

// Await parks until Release(t) for this world. This is the instrumentation point.
func (c *Controller) Await(t scenario.Transition) {
	key := t.String()
	c.mu.Lock()
	ch := c.ensureLocked(key)
	c.parked[key]++
	c.broadcastLocked()
	c.mu.Unlock()

	<-ch

	c.mu.Lock()
	c.parked[key]--
	c.broadcastLocked()
	c.mu.Unlock()
}

// Release opens transition t and waits until the waiter has left that Await.
// Each transition is one-shot per Controller.
func (c *Controller) Release(t scenario.Transition) {
	key := t.String()
	c.mu.Lock()
	ch := c.ensureLocked(key)
	select {
	case <-ch:
		// already closed
	default:
		close(ch)
	}
	c.broadcastLocked()
	c.mu.Unlock()

	// Must wait for the parked goroutine to observe the close. Otherwise a
	// tight Release/Quiesce loop on the explorer goroutine can starve workers
	// and Quiesce stays "settled" on stale park counts.
	for {
		c.mu.Lock()
		if c.parked[key] == 0 {
			c.mu.Unlock()
			return
		}
		wait := make(chan struct{})
		c.waiters = append(c.waiters, wait)
		c.mu.Unlock()
		<-wait
	}
}

// Quiesce blocks until every tracked goroutine has either exited or parked in Await.
func (c *Controller) Quiesce() {
	for {
		c.mu.Lock()
		if c.settledLocked() {
			c.mu.Unlock()
			return
		}
		ch := make(chan struct{})
		c.waiters = append(c.waiters, ch)
		c.mu.Unlock()
		<-ch
	}
}

func (c *Controller) settledLocked() bool {
	if c.living == 0 {
		return true
	}
	active := c.living - c.exited
	if active == 0 {
		return true
	}
	parked := 0
	for _, n := range c.parked {
		parked += n
	}
	return parked == active
}

func (c *Controller) ensureLocked(key string) chan struct{} {
	ch, ok := c.release[key]
	if !ok {
		ch = make(chan struct{})
		c.release[key] = ch
	}
	return ch
}

func (c *Controller) broadcastLocked() {
	for _, ch := range c.waiters {
		close(ch)
	}
	c.waiters = nil
}

// Program is a real-Go concurrent program instrumented with Await points.
type Program struct {
	Workers []scenario.WorkerID
	Steps   int // cooperative steps per worker (0 .. Steps-1)
	// Start launches workers via c.Go. Each worker must Await every
	// Transition{worker, step} for step in 0..Steps-1 in order.
	Start func(c *Controller)
	// Observe returns invariant-facing state after Quiesce.
	Observe func(c *Controller) scenario.State
}

func (p Program) validate() error {
	if p.Steps < 1 {
		return fmt.Errorf("gate: Steps must be >= 1")
	}
	if len(p.Workers) < 1 {
		return fmt.Errorf("gate: need at least one worker")
	}
	if p.Start == nil || p.Observe == nil {
		return fmt.Errorf("gate: Start and Observe required")
	}
	return nil
}

func enabled(next map[scenario.WorkerID]int, workers []scenario.WorkerID, steps int) []scenario.Transition {
	var out []scenario.Transition
	for _, w := range workers {
		if next[w] < steps {
			out = append(out, scenario.Transition{Worker: w, Step: next[w]})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Worker != out[j].Worker {
			return out[i].Worker < out[j].Worker
		}
		return out[i].Step < out[j].Step
	})
	return out
}

func copyNext(next map[scenario.WorkerID]int) map[scenario.WorkerID]int {
	out := make(map[scenario.WorkerID]int, len(next))
	for k, v := range next {
		out[k] = v
	}
	return out
}
