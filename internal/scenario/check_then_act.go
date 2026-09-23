package scenario

// CheckThenAct models a TOCTOU claim race on a shared resource.
//
// Two workers, two steps each:
//
//	0 check — copy Taken into a private local (true means already claimed)
//	1 act   — if local was false, set Taken and increment Owners
//
// Invariant: Owners <= 1. Some interleavings let both workers claim.
type CheckThenAct struct{}

// CheckThenActState is the snapshot of a CheckThenAct run.
type CheckThenActState struct {
	Taken  bool
	Owners int
	LocalA bool
	LocalB bool
	NextA  int
	NextB  int
}

// Initial implements Scenario.
func (CheckThenAct) Initial() State {
	return CheckThenActState{}
}

// Enabled implements Scenario.
func (CheckThenAct) Enabled(st State) []Transition {
	s := st.(CheckThenActState)
	var out []Transition
	if s.NextA < 2 {
		out = append(out, Transition{Worker: "A", Step: s.NextA})
	}
	if s.NextB < 2 {
		out = append(out, Transition{Worker: "B", Step: s.NextB})
	}
	return out
}

// Apply implements Scenario.
func (CheckThenAct) Apply(st State, t Transition) State {
	s := st.(CheckThenActState)
	switch t.Worker {
	case "A":
		s = applyCheckAct(s, t.Step, true)
	case "B":
		s = applyCheckAct(s, t.Step, false)
	default:
		panic("scenario: unknown worker " + string(t.Worker))
	}
	return s
}

func applyCheckAct(s CheckThenActState, step int, isA bool) CheckThenActState {
	var local *bool
	var next *int
	if isA {
		local, next = &s.LocalA, &s.NextA
	} else {
		local, next = &s.LocalB, &s.NextB
	}
	switch step {
	case 0:
		*local = s.Taken
	case 1:
		if !*local {
			s.Taken = true
			s.Owners++
		}
	default:
		panic("scenario: unknown step")
	}
	*next = step + 1
	return s
}
