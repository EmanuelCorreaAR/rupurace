package gate

import (
	"fmt"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Explore finds the first schedule (deterministic order) that violates a spec,
// by executing each candidate prefix on a fresh real-Go world.
func Explore(p Program, specs []invariant.Spec) (witness.Result, error) {
	if err := p.validate(); err != nil {
		return witness.Result{}, err
	}
	next := make(map[scenario.WorkerID]int, len(p.Workers))
	for _, w := range p.Workers {
		next[w] = 0
	}
	found, explored := search(p, specs, next, nil, 0)
	if found != nil {
		return witness.Result{
			Passed:   false,
			Steps:    len(found.Schedule),
			Schedule: found.Schedule,
			Witness:  found,
			Explored: explored,
			Strategy: "gate_exhaustive",
		}, nil
	}
	return witness.Result{
		Passed:   true,
		Explored: explored,
		Strategy: "gate_exhaustive",
	}, nil
}

func search(
	p Program,
	specs []invariant.Spec,
	next map[scenario.WorkerID]int,
	sched scenario.Schedule,
	explored int,
) (*witness.Witness, int) {
	en := enabled(next, p.Workers, p.Steps)
	if len(en) == 0 {
		return nil, explored + 1
	}
	for _, t := range en {
		cand := append(append(scenario.Schedule{}, sched...), t)
		w, err := execute(p, specs, cand)
		explored++
		if err != nil {
			// Illegal relative to park protocol — skip (should not happen with DFS).
			continue
		}
		if w != nil {
			return w, explored
		}
		n2 := copyNext(next)
		n2[t.Worker]++
		if got, n := search(p, specs, n2, cand, explored); got != nil {
			return got, n
		} else {
			explored = n
		}
	}
	return nil, explored
}

// Replay re-executes schedule on a fresh world (same contract as replay.Run).
func Replay(p Program, specs []invariant.Spec, sched scenario.Schedule) (witness.Result, error) {
	if err := p.validate(); err != nil {
		return witness.Result{}, err
	}
	w, err := execute(p, specs, sched)
	if err != nil {
		return witness.Result{}, err
	}
	if w != nil {
		return witness.Result{
			Passed:   false,
			Steps:    len(w.Schedule),
			Schedule: w.Schedule,
			Witness:  w,
			Strategy: "gate_replay",
		}, nil
	}
	return witness.Result{
		Passed:   true,
		Steps:    len(sched),
		Schedule: sched,
		Strategy: "gate_replay",
	}, nil
}

func execute(p Program, specs []invariant.Spec, sched scenario.Schedule) (*witness.Witness, error) {
	c := NewController()
	p.Start(c)
	c.Quiesce()

	built := make(scenario.Schedule, 0, len(sched))
	next := make(map[scenario.WorkerID]int, len(p.Workers))
	for _, w := range p.Workers {
		next[w] = 0
	}

	for i, t := range sched {
		en := enabled(next, p.Workers, p.Steps)
		if !contains(en, t) {
			return nil, fmt.Errorf("gate: step %d transition %s not enabled", i, t)
		}
		c.Release(t)
		c.Quiesce()
		built = append(built, t)
		next[t.Worker]++

		obs := p.Observe(c)
		if got := invariant.Check(obs, built, specs); got != nil {
			return got, nil
		}
	}
	return nil, nil
}

func contains(en []scenario.Transition, want scenario.Transition) bool {
	for _, t := range en {
		if t == want {
			return true
		}
	}
	return false
}
