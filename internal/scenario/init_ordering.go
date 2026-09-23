package scenario

// InitOrdering models a publish-before-write initialization bug.
//
// Publisher (P) incorrectly publishes before writing the payload:
//
//	0 publish — set Published = true
//	1 write   — set Payload = Expected
//
// Subscriber (S):
//
//	0 observe — copy Published into local
//	1 read    — if local, copy Payload into Got
//
// Invariant: Got == 0 || Got == Expected.
// Some interleavings let the subscriber observe Published and read a zero payload.
type InitOrdering struct {
	Expected int
}

// InitOrderingState is the snapshot of an InitOrdering run.
type InitOrderingState struct {
	Published bool
	Payload   int
	Got       int
	LocalS    bool
	NextP     int
	NextS     int
}

// Initial implements Scenario.
func (i InitOrdering) Initial() State {
	return InitOrderingState{}
}

// Enabled implements Scenario.
func (InitOrdering) Enabled(st State) []Transition {
	s := st.(InitOrderingState)
	var out []Transition
	if s.NextP < 2 {
		out = append(out, Transition{Worker: "P", Step: s.NextP})
	}
	if s.NextS < 2 {
		out = append(out, Transition{Worker: "S", Step: s.NextS})
	}
	return out
}

// Apply implements Scenario.
func (i InitOrdering) Apply(st State, t Transition) State {
	s := st.(InitOrderingState)
	switch t.Worker {
	case "P":
		switch t.Step {
		case 0:
			s.Published = true
		case 1:
			s.Payload = i.Expected
		default:
			panic("scenario: unknown step")
		}
		s.NextP = t.Step + 1
	case "S":
		switch t.Step {
		case 0:
			s.LocalS = s.Published
		case 1:
			if s.LocalS {
				s.Got = s.Payload
			}
		default:
			panic("scenario: unknown step")
		}
		s.NextS = t.Step + 1
	default:
		panic("scenario: unknown worker " + string(t.Worker))
	}
	return s
}
