package gate

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

// Sentinel errors for adversarial / malformed worlds.
var (
	// ErrNoWaiter means Release was called while no goroutine is parked at that
	// transition (early return, skipped gate, or schedule ahead of the SUT).
	ErrNoWaiter = errors.New("gate: no waiter at transition")
	// ErrAlreadyReleased means the one-shot gate was opened twice.
	ErrAlreadyReleased = errors.New("gate: transition already released")
)

// Controller mediates named transitions between the SUT and the explorer.
type Controller struct {
	mu       sync.Mutex
	release  map[string]chan struct{}
	parked   map[string]int
	living   int
	exited   int
	waiters  []chan struct{}
	stash    any // per-world demo state (internal)
	panicked []any
}

// NewController returns a fresh world controller.
func NewController() *Controller {
	return &Controller{
		release: make(map[string]chan struct{}),
		parked:  make(map[string]int),
	}
}

// Go starts a goroutine tracked for Quiesce. Panics inside fn are recovered so
// one worker cannot abort the whole Explore process; see Panicked.
func (c *Controller) Go(fn func()) {
	c.mu.Lock()
	c.living++
	c.mu.Unlock()
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				c.mu.Lock()
				c.panicked = append(c.panicked, rec)
				c.mu.Unlock()
			}
			c.mu.Lock()
			c.exited++
			c.broadcastLocked()
			c.mu.Unlock()
		}()
		fn()
	}()
}

// Panicked returns recover() values from worker goroutines (copy).
func (c *Controller) Panicked() []any {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]any, len(c.panicked))
	copy(out, c.panicked)
	return out
}

// ParkedKeys returns transitions that currently have waiters (sorted).
func (c *Controller) ParkedKeys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var keys []string
	for k, n := range c.parked {
		if n > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
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
//
// Contract: call Quiesce first. Release returns ErrNoWaiter if nobody is parked
// at t (skipped gate / early exit / wrong schedule). Returns ErrAlreadyReleased
// on a second open of the same transition.
func (c *Controller) Release(t scenario.Transition) error {
	key := t.String()
	c.mu.Lock()
	ch, exists := c.release[key]
	if exists {
		select {
		case <-ch:
			c.mu.Unlock()
			return fmt.Errorf("%w: %s", ErrAlreadyReleased, key)
		default:
		}
	} else {
		ch = make(chan struct{})
		c.release[key] = ch
	}
	if c.parked[key] == 0 {
		c.mu.Unlock()
		return fmt.Errorf("%w: %s (parked=%v)", ErrNoWaiter, key, c.parkedKeysLocked())
	}
	close(ch)
	c.broadcastLocked()
	c.mu.Unlock()

	// Wait for the parked goroutine to observe the close (avoid explorer starvation).
	for {
		c.mu.Lock()
		if c.parked[key] == 0 {
			c.mu.Unlock()
			return nil
		}
		wait := make(chan struct{})
		c.waiters = append(c.waiters, wait)
		c.mu.Unlock()
		<-wait
	}
}

func (c *Controller) parkedKeysLocked() []string {
	var keys []string
	for k, n := range c.parked {
		if n > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// Quiesce blocks until every tracked goroutine has either exited or parked in Await.
//
// Limitation: a goroutine blocked on something other than Await (mutex, channel
// receive, I/O, …) is not “parked” — Quiesce will not return. That is deadlock
// from the explorer’s point of view, not quiescence.
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
