package gate

import (
	"fmt"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

// OwnersState is the observable for the live check-then-act demo.
type OwnersState struct {
	Owners int
}

type ctaShared struct {
	available bool
	owners    int
}

// CheckThenAct returns a Program: real goroutines + Await gates, same bug as
// scenario.CheckThenAct — logic stays in ordinary Go, not Apply tables.
func CheckThenAct() Program {
	// Shared heap so Start and Observe close over the same world.
	// A new Program value is created per Explore/Replay call via factory...
	// Actually Program.Start is called once per world; we need fresh shared
	// per Start. Build shared inside Start and stash on Controller.
	return Program{
		Workers: []scenario.WorkerID{"A", "B"},
		Steps:   2,
		Start: func(c *Controller) {
			s := &ctaShared{available: true}
			c.stash = s
			run := func(w scenario.WorkerID) {
				c.Await(scenario.Transition{Worker: w, Step: 0})
				ok := s.available
				c.Await(scenario.Transition{Worker: w, Step: 1})
				if ok {
					s.available = false
					s.owners++
				}
			}
			c.Go(func() { run("A") })
			c.Go(func() { run("B") })
		},
		Observe: func(c *Controller) scenario.State {
			s, _ := c.stash.(*ctaShared)
			if s == nil {
				return OwnersState{}
			}
			return OwnersState{Owners: s.owners}
		},
	}
}

// OwnersSpec is the classic TOCTOU invariant.
func OwnersSpec() []invariant.Spec {
	return []invariant.Spec{{
		Name: "owners <= 1",
		Hold: func(st scenario.State) error {
			owners := ownersOf(st)
			if owners > 1 {
				return fmt.Errorf("owners = %d", owners)
			}
			return nil
		},
	}}
}

func ownersOf(st scenario.State) int {
	switch s := st.(type) {
	case OwnersState:
		return s.Owners
	case *liveState:
		if o, ok := s.obs.(OwnersState); ok {
			return o.Owners
		}
		return 0
	default:
		return 0
	}
}
