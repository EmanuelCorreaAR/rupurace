package gate

import (
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

// ReplayScenario adapts a Program to scenario.Scenario for linear Replay and
// Minimize only. Each Initial() starts a fresh world. Do not use with
// ExploreExhaustive — Apply mutates a live Controller and cannot fork.
func ReplayScenario(p Program) scenario.Scenario {
	return replayScenario{p: p}
}

type replayScenario struct {
	p Program
}

type liveState struct {
	c    *Controller
	next map[scenario.WorkerID]int
	obs  scenario.State
	p    Program
}

func (r replayScenario) Initial() scenario.State {
	c := NewController()
	r.p.Start(c)
	c.Quiesce()
	next := make(map[scenario.WorkerID]int, len(r.p.Workers))
	for _, w := range r.p.Workers {
		next[w] = 0
	}
	return &liveState{c: c, next: next, obs: r.p.Observe(c), p: r.p}
}

func (r replayScenario) Enabled(st scenario.State) []scenario.Transition {
	s := st.(*liveState)
	return enabled(s.next, s.p.Workers, s.p.Steps)
}

func (r replayScenario) Apply(st scenario.State, t scenario.Transition) scenario.State {
	s := st.(*liveState)
	s.c.Release(t)
	s.c.Quiesce()
	n2 := copyNext(s.next)
	n2[t.Worker]++
	return &liveState{
		c:    s.c,
		next: n2,
		obs:  s.p.Observe(s.c),
		p:    s.p,
	}
}
