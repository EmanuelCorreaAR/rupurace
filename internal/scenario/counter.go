package scenario

// Counter is a tiny abstract concurrent program:
// two workers each increment a shared counter up to their quota.
// Each increment is one cooperative step (index = how many times that worker has run).
type Counter struct {
	QuotaA int
	QuotaB int
}

// CounterState is the snapshot of a Counter run.
type CounterState struct {
	ADone int
	BDone int
	Value int
}

// Initial implements Scenario.
func (c Counter) Initial() State {
	return CounterState{}
}

// Enabled implements Scenario.
func (c Counter) Enabled(st State) []Transition {
	cs := st.(CounterState)
	var out []Transition
	if cs.ADone < c.QuotaA {
		out = append(out, Transition{Worker: "A", Step: cs.ADone})
	}
	if cs.BDone < c.QuotaB {
		out = append(out, Transition{Worker: "B", Step: cs.BDone})
	}
	return out
}

// Apply implements Scenario.
func (c Counter) Apply(st State, t Transition) State {
	cs := st.(CounterState)
	switch t.Worker {
	case "A":
		cs.ADone++
		cs.Value++
	case "B":
		cs.BDone++
		cs.Value++
	default:
		panic("scenario: unknown worker " + string(t.Worker))
	}
	return cs
}
