package scenario

// LostUpdate models a classic read-modify-write race on a shared counter.
//
// Two workers, three steps each:
//
//	0 read  — copy Value into a private local
//	1 inc   — local++
//	2 write — store local back into Value
//
// Expected final Value is 2. Some interleavings lose an update (final Value is 1).
type LostUpdate struct{}

// LostUpdateState is the snapshot of a LostUpdate run.
type LostUpdateState struct {
	Value  int
	LocalA int
	LocalB int
	NextA  int
	NextB  int
}

// Initial implements Scenario.
func (LostUpdate) Initial() State {
	return LostUpdateState{}
}

// Enabled implements Scenario.
func (LostUpdate) Enabled(st State) []Transition {
	s := st.(LostUpdateState)
	var out []Transition
	if s.NextA < 3 {
		out = append(out, Transition{Worker: "A", Step: s.NextA})
	}
	if s.NextB < 3 {
		out = append(out, Transition{Worker: "B", Step: s.NextB})
	}
	return out
}

// Apply implements Scenario.
func (LostUpdate) Apply(st State, t Transition) State {
	s := st.(LostUpdateState)
	switch t.Worker {
	case "A":
		s = applyLostStep(s, t.Step, true)
	case "B":
		s = applyLostStep(s, t.Step, false)
	default:
		panic("scenario: unknown worker " + string(t.Worker))
	}
	return s
}

func applyLostStep(s LostUpdateState, step int, isA bool) LostUpdateState {
	var local *int
	var next *int
	if isA {
		local, next = &s.LocalA, &s.NextA
	} else {
		local, next = &s.LocalB, &s.NextB
	}
	switch step {
	case 0:
		*local = s.Value
	case 1:
		*local = *local + 1
	case 2:
		s.Value = *local
	default:
		panic("scenario: unknown step")
	}
	*next = step + 1
	return s
}

// Done reports whether both workers finished.
func (s LostUpdateState) Done() bool {
	return s.NextA >= 3 && s.NextB >= 3
}
